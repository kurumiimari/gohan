package wallet

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/client"
	"github.com/kurumiimari/gohan/log"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/pkg/errors"
	"gopkg.in/tomb.v2"
	"sync"
	"time"
)

const (
	BlockMonitorFinalityDepth = 10
)

var (
	bmLogger = log.ModuleLogger("block-monitor")

	ErrBlockMonitorSafetyStop = errors.New("block monitor safety stop")
)

type BlockMonitor struct {
	tmb         *tomb.Tomb
	client      *client.NodeRPCClient
	engine      *walletdb.Engine
	subs        []chan *BlockNotification
	checkpoints []*walletdb.BlockCheckpoint
	lastHeight  int
	mtx         sync.RWMutex
	dead        bool
}

type BlockNotification struct {
	ChainTip  int
	CommonTip int
}

func NewBlockMonitor(tmb *tomb.Tomb, client *client.NodeRPCClient, engine *walletdb.Engine) *BlockMonitor {
	return &BlockMonitor{
		tmb:    tmb,
		client: client,
		engine: engine,
	}
}

func (b *BlockMonitor) Start() error {
	var checkpoints []*walletdb.BlockCheckpoint
	err := b.engine.Transaction(func(tx walletdb.Transactor) error {
		checks, err := walletdb.GetBlockCheckpoints(tx)
		if err != nil {
			return err
		}
		checkpoints = checks
		return nil
	})
	if err != nil {
		return err
	}
	b.checkpoints = checkpoints

	b.tmb.Go(func() error {
		if err := b.poll(); err != nil {
			bmLogger.Error("error polling", "err", err)
			return err
		}

		tick := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-tick.C:
				if err := b.poll(); err != nil {
					bmLogger.Error("error polling", "err", err)
				}
			case <-b.tmb.Dying():
				b.mtx.Lock()
				b.dead = true
				for _, sub := range b.subs {
					close(sub)
				}
				b.mtx.Unlock()
				return nil
			}
		}
	})

	return nil
}

func (b *BlockMonitor) LastHeight() int {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return b.lastHeight
}

func (b *BlockMonitor) Subscribe() <-chan *BlockNotification {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if b.dead {
		panic("block monitor is closed")
	}

	ch := make(chan *BlockNotification, 1)
	b.subs = append(b.subs, ch)
	return ch
}

func (b *BlockMonitor) Poll() error {
	return b.poll()
}

func (b *BlockMonitor) poll() error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if b.dead {
		panic("block monitor is dead")
	}

	info, err := b.client.GetInfo()
	if err != nil {
		return errors.Wrap(err, "error getting block height")
	}

	if info.Blocks == 0 {
		return nil
	}

	if len(b.checkpoints) == 0 {
		if err := b.updateBlockCheckpoints(info.Blocks); err != nil {
			return err
		}
		return nil
	}

	checkTip := b.checkpoints[len(b.checkpoints)-1]

	// this shouldn't happen
	if checkTip.Height > info.Blocks {
		return ErrBlockMonitorSafetyStop
	}

	// if we expect the checkpoint block to be "finalized", just make sure its hash matches
	if info.Blocks-checkTip.Height > BlockMonitorFinalityDepth {
		blockB, err := b.client.GetRawBlock(checkTip.Height)
		if err != nil {
			return err
		}
		block, err := chain.NewBlockFromBytes(blockB)
		if err != nil {
			return err
		}

		if block.HashHex() != checkTip.Hash {
			bmLogger.Error(
				"deep reorg detected",
				"chain_height",
				b.lastHeight,
				"checkpoint_height",
				checkTip.Height,
				"chain_hash",
				block.HashHex(),
				"checkpoint_hash",
				checkTip.Hash,
			)
			return ErrBlockMonitorSafetyStop
		}

		if err := b.updateBlockCheckpoints(info.Blocks); err != nil {
			return err
		}

		b.lastHeight = info.Blocks
		b.sendNotifications(b.lastHeight, b.lastHeight)
		return nil
	}

	blocks, heights, err := b.getBlocksDescending(b.checkpoints[0].Height, BlockMonitorFinalityDepth)
	if err != nil {
		return err
	}

	chainBlocksByHeight := make(map[int]*chain.Block)
	for i, block := range blocks {
		chainBlocksByHeight[heights[i]] = block
	}

	// determine first block in common with
	// our checkpoints
	commonBlock := -1
	for _, check := range b.checkpoints {
		if chainBlocksByHeight[check.Height].HashHex() == check.Hash {
			commonBlock = check.Height
			break
		}
	}
	if commonBlock == -1 {
		return ErrBlockMonitorSafetyStop
	}

	var commonTip int
	if commonBlock == b.checkpoints[0].Height {
		// if the common block is the highest checkpoint
		// block, return lastHeight since the known blocks
		// have not reorged
		commonTip = info.Blocks
	} else {
		// looks like we have a reorg. roll back
		commonTip = commonBlock
	}

	if err := b.updateBlockCheckpoints(info.Blocks); err != nil {
		return err
	}
	b.lastHeight = info.Blocks
	b.sendNotifications(b.lastHeight, commonTip)
	return nil
}

func (b *BlockMonitor) updateBlockCheckpoints(newHeight int) error {
	blocks, heights, err := b.getBlocksDescending(newHeight, 10)

	var checkpoints []*walletdb.BlockCheckpoint
	for i, block := range blocks {
		height := heights[i]
		checkpoints = append(checkpoints, &walletdb.BlockCheckpoint{
			Height: height,
			Hash:   block.HashHex(),
		})
	}

	err = b.engine.Transaction(func(tx walletdb.Transactor) error {
		return walletdb.UpdateBlockCheckpoints(tx, checkpoints)
	})
	if err != nil {
		return err
	}

	for i, j := 0, len(checkpoints)-1; i < j; i, j = i+1, j-1 {
		checkpoints[i], checkpoints[j] = checkpoints[j], checkpoints[i]
	}

	b.checkpoints = checkpoints
	return nil
}

func (b *BlockMonitor) sendNotifications(chainTip int, commonTip int) {
	notif := &BlockNotification{
		ChainTip:  chainTip,
		CommonTip: commonTip,
	}
	subsCopy := make([]chan *BlockNotification, len(b.subs))
	copy(subsCopy, b.subs)
	for _, sub := range subsCopy {
		sub <- notif
	}
}

func (b *BlockMonitor) getBlocksDescending(height int, count int) ([]*chain.Block, []int, error) {
	startBlock := height + 1 - count
	if startBlock < 1 {
		startBlock = 1
		count = height
	}

	chainBlocks, err := b.client.GetRawBlocksBatch(startBlock, count)
	if err != nil {
		return nil, nil, err
	}

	var blocks []*chain.Block
	var heights []int
	for i, cb := range chainBlocks {
		if cb.Error != nil {
			return nil, nil, errors.Wrap(cb.Error, "error getting block")
		}
		block, err := chain.NewBlockFromBytes(cb.Data)
		if err != nil {
			return nil, nil, err
		}
		blocks = append(blocks, block)
		heights = append(heights, startBlock+i)
	}
	return blocks, heights, nil
}
