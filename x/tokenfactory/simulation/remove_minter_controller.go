package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"noble/x/tokenfactory/keeper"
	"noble/x/tokenfactory/types"
)

func SimulateMsgRemoveMinterController(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgRemoveMinterController{
			From: simAccount.Address.String(),
		}

		// TODO: Handling the RemoveMinterController simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "RemoveMinterController simulation not implemented"), nil, nil
	}
}
