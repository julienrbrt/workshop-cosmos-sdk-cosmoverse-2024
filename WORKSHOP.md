# Workshop

## Introduction

Cosmos SDK v0.52+ introduces the new x/accounts module.
This module allows to chain developer to define smart accounts (instead of EOA) with custom logic.
By default, the SDK provides the following accounts: `BaseAccount`, `Multisig`, `Lockup`, `Authz`, `Feegrant`.

This allows to replace modules that were actually only account capabilities:

* x/auth baseaccount -> x/accounts base account
* x/group -> x/accounts multisig account
* x/auth/vesting -> x/accounts lockup account (a legally speaking vesting account should be implemented by the chain developers)
* x/authz -> x/accounts authz account
* x/feegrant -> x/accounts feegrant account

NOTE: The difference between interacting with an EOA compared to a x/accounts account, is that x/accounts actions must be performed via x/accounts.

This workshop will walk through how to implement such account.

## Smart Account

Today we'll build a secure multi-signature account that requires all signers to approve transactions within a fixed block window.

### Key Features

* All signers must approve within 20 blocks
* Any signer can initiate a transaction
* Transactions expire if not fully signed
* State tracking for pending transactions
