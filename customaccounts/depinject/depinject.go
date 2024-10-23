package customaccountdepinject

import (
	"cosmossdk.io/x/accounts/accountstd"

	customaccounts "github.com/cosmosregistry/chain-minimal/customaccounts"
)

func ProvideAccount() accountstd.DepinjectAccount {
	return accountstd.DIAccount("mutisigcosmoverse", customaccounts.NewAccount)
}
