package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v3"
	"github.com/strangelove-ventures/interchaintest/v3/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v3/ibc"
	"github.com/strangelove-ventures/interchaintest/v3/testreporter"
	"github.com/strangelove-ventures/noble/cmd"
	integration "github.com/strangelove-ventures/noble/interchaintest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestLoad(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx := context.Background()

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	client, network := interchaintest.DockerSetup(t)

	repo, version := integration.GetDockerImageInfo()

	var noble *cosmos.CosmosChain
	var roles NobleRoles
	var roles2 NobleRoles
	var paramauthorityWallet Authority
	var extraWallets ExtraWallets

	chainCfg := ibc.ChainConfig{
		Type:           "cosmos",
		Name:           "noble",
		ChainID:        "noble-1",
		Bin:            "nobled",
		Denom:          "utoken",
		Bech32Prefix:   "noble",
		CoinType:       "118",
		GasPrices:      "0.0token",
		GasAdjustment:  1.1,
		TrustingPeriod: "504h",
		NoHostMount:    false,
		Images: []ibc.DockerImage{
			{
				Repository: repo,
				Version:    version,
				UidGid:     "1025:1025",
			},
		},
		EncodingConfig: NobleEncoding(),
		PreGenesis: func(cc ibc.ChainConfig) error {
			val := noble.Validators[0]
			err := createTokenfactoryRoles(ctx, &roles, DenomMetadata_rupee, val, true)
			if err != nil {
				return err
			}
			err = createTokenfactoryRoles(ctx, &roles2, DenomMetadata_drachma, val, false)
			if err != nil {
				return err
			}
			extraWallets, err = createExtraWalletsAtGenesis(ctx, val)
			if err != nil {
				return err
			}
			paramauthorityWallet, err = createParamAuthAtGenesis(ctx, val)
			return err
		},
		ModifyGenesis: func(cc ibc.ChainConfig, b []byte) ([]byte, error) {
			g := make(map[string]interface{})
			if err := json.Unmarshal(b, &g); err != nil {
				return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
			}
			if err := modifyGenesisTokenfactory(g, "tokenfactory", DenomMetadata_rupee, &roles, true); err != nil {
				return nil, err
			}
			if err := modifyGenesisTokenfactory(g, "fiat-tokenfactory", DenomMetadata_drachma, &roles2, true); err != nil {
				return nil, err
			}
			if err := modifyGenesisParamAuthority(g, paramauthorityWallet.Authority.Address); err != nil {
				return nil, err
			}
			out, err := json.Marshal(&g)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
			}
			return out, nil
		},
	}

	nv := 2
	nf := 0

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			ChainConfig:   chainCfg,
			NumValidators: &nv,
			NumFullNodes:  &nf,
		},
	})

	cmd.SetPrefixes(chainCfg.Bech32Prefix)

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	noble = chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(noble)

	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,

		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	require.NoError(t, noble.CreateKey(ctx, "newWallet"), "error creating key")
	addressBytes, err := noble.GetAddress(ctx, "newWallet")
	require.NoError(t, err, "error getting wallet address")
	address, err := types.Bech32ifyAddressBytes(noble.Config().Bech32Prefix, addressBytes)
	require.NoError(t, err, "error converting address")

	bal, err := noble.GetBalance(ctx, string(extraWallets.User2.Address), noble.Config().Denom)
	require.NoError(t, err, "error getting balance")
	t.Log("BALANCE of USER 2: ", bal)

	// send amount
	send := ibc.WalletAmount{
		Address: string(address),
		Denom:   noble.Config().Denom,
		Amount:  10,
	}

	// duration := 5 * time.Second
	// startTime := time.Now()
	// endTime := startTime.Add(duration)

	// var counter int

	// for time.Now().Before(endTime) {
	// 	counter++
	// 	require.NoError(t, noble.SendFunds(ctx, extraWallets.User2.KeyName, send))
	// }

	// t.Log("Counter: ", counter)
	// testutil.WaitForBlocks(ctx, 5, noble)

	threadedFunc(30, func() {
		require.NoError(t, noble.SendFunds(ctx, extraWallets.User2.KeyName, send))
	})

	t.Log("BALANCE of new wallet after tx's!!!! ", bal)

	bal, err = noble.GetBalance(ctx, string(address), noble.Config().Denom)
	require.NoError(t, err, "error getting balance")
	t.Log("Final Balance: ", bal)

}

func threadedFunc(x int, y func()) {
	var wg sync.WaitGroup
	wg.Add(x)
	for i := 0; i < x; i++ {
		go func() {
			defer wg.Done()
			y()
		}()
	}

	wg.Wait()
}