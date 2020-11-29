package connmgr

import (
	"fmt"
	"github.com/multivactech/MultiVAC/model/chaincfg"
	"github.com/multivactech/MultiVAC/model/wire"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSeedFromDNS(t *testing.T) {
	params := chaincfg.Params{
		Name:        "Davis",
		Net:         0,
		DefaultPort: "2333",
		DNSSeeds: []chaincfg.DNSSeed{
			{
				Host:         "127.0.0.1",
				HasFiltering: false,
			}, {
				Host:         "210.31.32.128",
				HasFiltering: false,
			}, {
				Host:         "192.168.1.12",
				HasFiltering: false,
			}, {
				Host:         "230.230.230.230",
				HasFiltering: true,
			},
		},
		PowLimit:                      nil,
		PowLimitBits:                  0,
		BIP0034Height:                 0,
		BIP0065Height:                 0,
		BIP0066Height:                 0,
		CoinbaseMaturity:              0,
		SubsidyReductionInterval:      0,
		TargetTimespan:                0,
		TargetTimePerBlock:            0,
		RetargetAdjustmentFactor:      0,
		ReduceMinDifficulty:           false,
		MinDiffReductionTime:          0,
		GenerateSupported:             false,
		Checkpoints:                   nil,
		RuleChangeActivationThreshold: 0,
		MinerConfirmationWindow:       0,
		Deployments:                   [3]chaincfg.ConsensusDeployment{},
		RelayNonStdTxs:                false,
		Bech32HRPSegwit:               "",
		PubKeyHashAddrID:              0,
		ScriptHashAddrID:              0,
		PrivateKeyID:                  0,
		WitnessPubKeyHashAddrID:       0,
		WitnessScriptHashAddrID:       0,
		HDPrivateKeyID:                [4]byte{},
		HDPublicKeyID:                 [4]byte{},
		HDCoinType:                    0,
	}
	SeedFromDNS(&params, 0, lookupfun, seedfn)
	time.Sleep(4 * time.Second)
}

func seedfn(addrs []*wire.NetAddress) {
	for _, val := range addrs {
		fmt.Println(string(val.IP))
	}
}
func lookupfun(s string) ([]net.IP, error) {
	if s == "127.0.0.1" {
		return []net.IP{
			{
				127, 0, 0, 1,
			}, {
				127, 0, 0, 2,
			},
		}, nil
	}
	if strings.Contains(s, "x") {
		return nil, fmt.Errorf("error from test,host:%v", s)
	}
	return []net.IP{}, nil
}
