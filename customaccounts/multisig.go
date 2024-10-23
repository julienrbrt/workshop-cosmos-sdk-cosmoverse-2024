package accounts

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/x/accounts/accountstd"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/cosmosregistry/chain-minimal/customaccounts/types"
)

func (a Account) Init(ctx context.Context, msg *types.MsgInit) (*types.MsgInitResponse, error) {
	if len(msg.Signers) < 2 {
		return nil, errors.New("multisig must have at least 2 signers")
	}

	// Set signing window (default 20 blocks)
	window := uint64(20)
	if msg.SigningWindow > 0 {
		window = msg.SigningWindow
	}

	if err := a.SigningWindow.Set(ctx, window); err != nil {
		return nil, err
	}

	// Set required signers (all signers must approve)
	if err := a.RequiredSigners.Set(ctx, uint64(len(msg.Signers))); err != nil {
		return nil, err
	}

	// Store authorized signers
	for _, signer := range msg.Signers {
		addr, err := a.addressCodec.StringToBytes(signer)
		if err != nil {
			return nil, err
		}

		if err := a.Signers.Set(ctx, addr, true); err != nil {
			return nil, err
		}
	}

	return &types.MsgInitResponse{}, nil
}

func (a Account) SubmitTx(ctx context.Context, msg *types.MsgSubmitTx) (*types.MsgSubmitTxResponse, error) {
	// Verify sender is authorized
	sender := accountstd.Sender(ctx)

	isSigner, err := a.Signers.Get(ctx, sender)
	if err != nil || !isSigner {
		return nil, errors.New("unauthorized: sender is not a signer")
	}

	// Get new transaction ID
	txID, err := a.TxSequence.Next(ctx)
	if err != nil {
		return nil, err
	}

	senderStr, err := a.addressCodec.BytesToString(sender)
	if err != nil {
		return nil, err
	}

	currentBlock := uint64(a.headerService.HeaderInfo(ctx).Height)
	window, err := a.SigningWindow.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Store pending transaction
	pending := types.PendingTx{
		TransactionId:    txID,
		Transaction:      msg.Transaction,
		Approvals:        []string{senderStr}, // Initiator counts as first approval
		SubmittedAtBlock: currentBlock,
		ExpiresAtBlock:   currentBlock + window,
		InitiatedBy:      senderStr,
	}

	err = a.PendingTxs.Set(ctx, txID, pending)
	if err != nil {
		return nil, err
	}

	// Record initiator's approval
	if err = a.Approvals.Set(ctx, collections.Join(txID, sender), true); err != nil {
		return nil, err
	}

	return &types.MsgSubmitTxResponse{
		TransactionId: txID,
	}, nil
}

func (a Account) ApproveTx(ctx context.Context, msg *types.MsgApproveTx) (*types.MsgApproveTxResponse, error) {
	// Verify sender is authorized
	sender := accountstd.Sender(ctx)

	isSigner, err := a.Signers.Get(ctx, sender)
	if err != nil || !isSigner {
		return nil, errors.New("unauthorized: sender is not a signer")
	}

	// Get pending transaction
	pending, err := a.PendingTxs.Get(ctx, msg.TransactionId)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	// Check if already approved by this signer
	approved, err := a.Approvals.Get(ctx,
		collections.Join(msg.TransactionId, sender))
	if err == nil && approved {
		return nil, errors.New("already approved by this signer")
	}

	// Verify within signing window
	currentHeight := uint64(a.headerService.HeaderInfo(ctx).Height)

	if currentHeight > pending.ExpiresAtBlock {
		// Clean up expired transaction
		err = a.cleanupTransaction(ctx, msg.TransactionId)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("transaction expired")
	}

	// Record approval
	if err = a.Approvals.Set(ctx,
		collections.Join(msg.TransactionId, sender),
		true); err != nil {
		return nil, err
	}

	senderStr, err := a.addressCodec.BytesToString(sender)
	if err != nil {
		return nil, err
	}

	// Update approval count
	pending.Approvals = append(pending.Approvals, senderStr)
	err = a.PendingTxs.Set(ctx, msg.TransactionId, pending)
	if err != nil {
		return nil, err
	}

	// Check if we have all required approvals
	required, err := a.RequiredSigners.Get(ctx)
	if err != nil {
		return nil, err
	}

	if uint64(len(pending.Approvals)) == required {
		_, execErr := accountstd.ExecModuleAnys(ctx, []*codectypes.Any{pending.Transaction})

		// Clean up completed transaction
		if err := a.cleanupTransaction(ctx, msg.TransactionId); err != nil {
			return nil, err
		}

		return &types.MsgApproveTxResponse{
			Executed: true,
			Failed:   execErr == nil,
		}, nil
	}

	return &types.MsgApproveTxResponse{
		Executed: false,
	}, nil
}

func (a Account) cleanupTransaction(ctx context.Context, txID uint64) error {
	// Delete pending transaction
	if err := a.PendingTxs.Remove(ctx, txID); err != nil {
		return err
	}

	// Delete all approvals
	return a.Approvals.Clear(ctx, collections.NewPrefixedPairRange[uint64, []byte](txID))
}

func (a Account) QueryPendingTx(ctx context.Context, req *types.QueryPendingTx) (*types.QueryPendingTxResponse, error) {
	pending, err := a.PendingTxs.Get(ctx, req.TransactionId)
	if err != nil {
		return nil, err
	}

	approvals := []string{}
	err = a.Approvals.Walk(ctx,
		collections.NewPrefixedPairRange[uint64, []byte](req.TransactionId),
		func(key collections.Pair[uint64, []byte], approved bool) (bool, error) {
			if approved {
				addr, err := a.addressCodec.BytesToString(key.K2())
				if err != nil {
					return true, err
				}
				approvals = append(approvals, addr)
			}
			return false, nil
		})
	if err != nil {
		return nil, err
	}

	window, err := a.SigningWindow.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryPendingTxResponse{
		Transaction:      pending.Transaction,
		Approvals:        approvals,
		SubmittedAtBlock: pending.SubmittedAtBlock,
		ExpiresAtBlock:   pending.SubmittedAtBlock + window,
		InitiatedBy:      pending.InitiatedBy,
	}, nil
}
