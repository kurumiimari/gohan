package client

type NameStart struct {
	Reserved bool `json:"reserved"`
	Week     int  `json:"week"`
	Start    int  `json:"start"`
}

type NameInfo struct {
	Name     string `json:"name"`
	NameHash string `json:"nameHash"`
	State    string `json:"state"`
	Height   int    `json:"height"`
	Renewal  int    `json:"renewal"`
	Owner    struct {
		Hash  string `json:"hash"`
		Index int    `json:"index"`
	} `json:"owner"`
	Value      int    `json:"value"`
	Highest    int    `json:"highest"`
	Data       string `json:"data"`
	Transfer   int    `json:"transfer"`
	Revoked    int    `json:"revoked"`
	Claimed    int    `json:"claimed"`
	Renewals   int    `json:"renewals"`
	Registered bool   `json:"registered"`
	Expired    bool   `json:"expired"`
	Weak       bool   `json:"weak"`
	Stats      struct {
		OpenPeriodStart          int     `json:"openPeriodStart"`
		OpenPeriodEnd            int     `json:"openPeriodEnd"`
		BlocksUntilBidding       int     `json:"blocksUntilBidding"`
		HoursUntilBidding        float64 `json:"hoursUntilBidding"`
		LockupPeriodStart        int     `json:"lockupPeriodStart"`
		LockupPeriodEnd          int     `json:"lockupPeriodEnd"`
		BlocksUntilClosed        int     `json:"blocksUntilClosed"`
		HoursUntilClosed         float64 `json:"hoursUntilClosed"`
		BidPeriodStart           int     `json:"bidPeriodStart"`
		BidPeriodEnd             int     `json:"bidPeriodEnd"`
		BlocksUntilReveal        int     `json:"blocksUntilReveal"`
		HoursUntilReveal         float64 `json:"hoursUntilReveal"`
		RevealPeriodStart        int     `json:"revealPeriodStart"`
		RevealPeriodEnd          int     `json:"revealPeriodEnd"`
		BlocksUntilClose         int     `json:"blocksUntilClose"`
		HoursUntilClose          float64 `json:"hoursUntilClose"`
		RenewalPeriodStart       int     `json:"renewalPeriodStart"`
		RenewalPeriodEnd         int     `json:"renewalPeriodEnd"`
		BlocksUntilExpire        int     `json:"blocksUntilExpire"`
		DaysUntilExpire          float64 `json:"daysUntilExpire"`
		RevokePeriodStart        int     `json:"revokePeriodStart"`
		RevokePeriodEnd          int     `json:"revokePeriodEnd"`
		BlocksUntilReopen        int     `json:"blocksUntilReopen"`
		HoursUntilReopen         float64 `json:"hoursUntilReopen"`
		TransferLockupStart      int     `json:"transferLockupStart"`
		TransferLockupEnd        int     `json:"transferLockupEnd"`
		BlocksUntilValidFinalize int     `json:"blocksUntilValidFinalize"`
		HoursUntilValidFinalize  float64 `json:"hoursUntilValidFinalize"`
	} `json:"stats"`
}

type NameInfoRes struct {
	Start *NameStart `json:"start"`
	Info  *NameInfo  `json:"info"`
}
