package itest

import (
	"github.com/kurumiimari/gohan/wallet"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type NodeSuite struct {
	suite.Suite
	hsd     *HSD
	client  *api.Client
	cleanup func()
}

func (s *NodeSuite) SetupSuite() {
	s.hsd = startHSD()
}

func (s *NodeSuite) TearDownSuite() {
	s.hsd.Stop()
}

func (s *NodeSuite) SetupTest() {
	t := s.T()
	s.client, s.cleanup = startDaemon(t)
}

func (s *NodeSuite) TearDownTest() {
	s.cleanup()
}

func (s *NodeSuite) TestStatus() {
	t := s.T()
	status, err := s.client.Status()
	require.NoError(t, err)
	require.Equal(t, &wallet.NodeStatus{
		Status: "OK",
		Height: 0,
	}, status)

	require.NoError(t, s.hsd.Client.GenerateToAddress(7, ZeroRegtestAddr))
	require.NoError(t, s.client.PollBlock())
	status, err = s.client.Status()
	require.NoError(t, err)
	require.Equal(t, &wallet.NodeStatus{
		Status: "OK",
		Height: 7,
	}, status)
}

func TestNodeSuite(t *testing.T) {
	suite.Run(t, new(NodeSuite))
}
