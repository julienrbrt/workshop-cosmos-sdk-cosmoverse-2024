package accounts

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/header"
	"cosmossdk.io/x/accounts/accountstd"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmosregistry/chain-minimal/customaccounts/types"
)

var (
	PendingTxsPrefix      = collections.NewPrefix(0)
	ApprovalsPrefix       = collections.NewPrefix(2)
	SequencePrefix        = collections.NewPrefix(1)
	SignersPrefix         = collections.NewPrefix(3)
	SigningWindowPrefix   = collections.NewPrefix(4)
	RequiredSignersPrefix = collections.NewPrefix(5)
)

// Compile-time type assertions
var _ accountstd.Interface = (*Account)(nil)

// Account is a multisig account implementation.
// Note state management of this account could be optimized, but not done for the sake of this workshop.
type Account struct {
	addressCodec  address.Codec
	headerService header.Service

	// Track pending transactions
	PendingTxs collections.Map[uint64, types.PendingTx]
	// Track approvals per transaction
	Approvals collections.Map[collections.Pair[uint64, []byte], bool]
	// Sequence for transaction IDs
	TxSequence collections.Sequence
	// List of authorized signers
	Signers collections.Map[[]byte, bool]
	// Number of blocks to wait for all signatures
	SigningWindow collections.Item[uint64]
	// Number of required signers
	RequiredSigners collections.Item[uint64]
}

// NewAccount returns a new multisig account creator function.
func NewAccount(deps accountstd.Dependencies) (*Account, error) {
	return &Account{
		addressCodec:    deps.AddressCodec,
		headerService:   deps.Environment.HeaderService,
		PendingTxs:      collections.NewMap[uint64, types.PendingTx](deps.SchemaBuilder, PendingTxsPrefix, "pending_txs", collections.Uint64Key, codec.CollValue[types.PendingTx](deps.LegacyStateCodec)),
		Approvals:       collections.NewMap(deps.SchemaBuilder, ApprovalsPrefix, "approvals", collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), collections.BoolValue),
		TxSequence:      collections.NewSequence(deps.SchemaBuilder, SequencePrefix, "tx_sequence"),
		Signers:         collections.NewMap(deps.SchemaBuilder, SignersPrefix, "signers", collections.BytesKey, collections.BoolValue),
		SigningWindow:   collections.NewItem[uint64](deps.SchemaBuilder, SigningWindowPrefix, "signing_window", collections.Uint64Value),
		RequiredSigners: collections.NewItem[uint64](deps.SchemaBuilder, RequiredSignersPrefix, "required_signers", collections.Uint64Value),
	}, nil
}

// RegisterExecuteHandlers implements implementation.Account.
func (a *Account) RegisterExecuteHandlers(builder *accountstd.ExecuteBuilder) {
	accountstd.RegisterExecuteHandler(builder, a.SubmitTx)
	accountstd.RegisterExecuteHandler(builder, a.ApproveTx)
}

// RegisterInitHandler implements implementation.Account.
func (a *Account) RegisterInitHandler(builder *accountstd.InitBuilder) {
	accountstd.RegisterInitHandler(builder, a.Init)
}

// RegisterQueryHandlers implements implementation.Account.
func (a *Account) RegisterQueryHandlers(builder *accountstd.QueryBuilder) {
	accountstd.RegisterQueryHandler(builder, a.QueryPendingTx)
}
