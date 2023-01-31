/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/onflow/flow-cli/pkg/flowkit/config"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-cli/pkg/flowkit/gateway"
	"github.com/onflow/flow-cli/pkg/flowkit/output"
	"github.com/onflow/flow-cli/pkg/flowkit/services"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
)

//go:generate go run github.com/vektra/mockery/cmd/mockery --name flowClient --filename mock_flow_test.go --inpkg
type flowClient interface {
	Initialize(configPath string, numberOfAccounts int) error
	GetClientAccount(name string) *clientAccount
	GetActiveClientAccount() *clientAccount
	GetClientAccounts() []*clientAccount
	SetActiveClientAccount(name string) error
	ExecuteScript(location *url.URL, args []cadence.Value) (cadence.Value, error)
	DeployContract(address flow.Address, name string, location *url.URL) error
	SendTransaction(
		authorizers []flow.Address,
		location *url.URL,
		args []cadence.Value,
	) (*flow.Transaction, *flow.TransactionResult, error)
	GetAccount(address flow.Address) (*flow.Account, error)
	CreateAccount() (*clientAccount, error)
}

var _ flowClient = &flowkitClient{}

type clientAccount struct {
	*flow.Account
	Name   string
	Active bool
}

var names = []string{
	"Alice", "Bob", "Charlie",
	"Dave", "Eve", "Faythe",
	"Grace", "Heidi", "Ivan",
	"Judy", "Michael", "Niaj",
	"Olivia", "Oscar", "Peggy",
	"Rupert", "Sybil", "Ted",
	"Victor", "Walter",
}

type flowkitClient struct {
	services      *services.Services
	loader        flowkit.ReaderWriter
	state         *flowkit.State
	accounts      []*clientAccount
	activeAccount *clientAccount
	configPath    string
}

func newFlowkitClient(loader flowkit.ReaderWriter) *flowkitClient {
	return &flowkitClient{
		loader: loader,
	}
}

func (f *flowkitClient) Initialize(configPath string, numberOfAccounts int) error {
	f.configPath = configPath
	state, err := flowkit.Load([]string{configPath}, f.loader)
	if err != nil {
		return err
	}
	f.state = state

	logger := output.NewStdoutLogger(output.NoneLog)

	serviceAccount, err := state.EmulatorServiceAccount()
	if err != nil {
		return err
	}

	var emulator gateway.Gateway
	// try connecting to already running local emulator
	emulator, err = gateway.NewGrpcGateway(config.DefaultEmulatorNetwork().Host)
	if err != nil || emulator.Ping() != nil { // fallback to hosted emulator if error
		emulator = gateway.NewEmulatorGateway(serviceAccount)
	}

	f.services = services.NewServices(emulator, state, logger)
	if numberOfAccounts > len(names) || numberOfAccounts <= 0 {
		return fmt.Errorf(fmt.Sprintf("only possible to create between 1 and %d accounts", len(names)))
	}

	f.accounts = make([]*clientAccount, 0)
	for i := 0; i < numberOfAccounts; i++ {
		_, err := f.CreateAccount()
		if err != nil {
			return err
		}
	}

	f.accounts[0].Active = true // make first active by default
	f.activeAccount = f.accounts[0]

	return nil
}

func (f *flowkitClient) GetClientAccount(name string) *clientAccount {
	for _, account := range f.accounts {
		if account.Name == name {
			return account
		}
	}
	return nil
}

func (f *flowkitClient) GetClientAccounts() []*clientAccount {
	return f.accounts
}

func (f *flowkitClient) SetActiveClientAccount(name string) error {
	activeAcc := f.GetActiveClientAccount()
	if activeAcc != nil {
		activeAcc.Active = false
	}

	account := f.GetClientAccount(name)
	if account == nil {
		return fmt.Errorf(fmt.Sprintf("account with a name %s not found", name))
	}

	account.Active = true
	f.activeAccount = account
	return nil
}

func (f *flowkitClient) GetActiveClientAccount() *clientAccount {
	return f.activeAccount
}

func (f *flowkitClient) ExecuteScript(
	location *url.URL,
	args []cadence.Value,
) (cadence.Value, error) {
	code, err := f.loader.ReadFile(location.Path)
	if err != nil {
		return nil, err
	}

	codeFilename, err := resolveFilename(f.configPath, location.Path)
	if err != nil {
		return nil, err
	}

	return f.services.Scripts.Execute(
		&services.Script{
			Code:     code,
			Args:     args,
			Filename: codeFilename,
		},
		config.DefaultEmulatorNetwork().Name,
	)
}

func (f *flowkitClient) DeployContract(
	address flow.Address,
	name string,
	location *url.URL,
) error {
	code, err := f.loader.ReadFile(location.Path)
	if err != nil {
		return err
	}

	service, err := f.state.EmulatorServiceAccount()
	if err != nil {
		return err
	}

	flowAccount, err := f.services.Accounts.Get(address)
	if err != nil {
		return err
	}

	// check if account already has a contract with this name deployed then update
	updateExisting := slices.Contains(maps.Keys(flowAccount.Contracts), name)

	codeFilename, err := resolveFilename(f.configPath, location.Path)
	if err != nil {
		return err
	}

	_, err = f.services.Accounts.AddContract(
		createSigner(address, service),
		&services.Contract{
			Script: &services.Script{
				Code:     code,
				Filename: codeFilename,
			},
			Network: config.DefaultEmulatorNetwork().Name,
		},
		updateExisting,
	)
	return err
}

func (f *flowkitClient) SendTransaction(
	authorizers []flow.Address,
	location *url.URL,
	args []cadence.Value,
) (*flow.Transaction, *flow.TransactionResult, error) {
	code, err := f.loader.ReadFile(location.Path)
	if err != nil {
		return nil, nil, err
	}

	service, err := f.state.EmulatorServiceAccount()
	if err != nil {
		return nil, nil, err
	}

	codeFilename, err := resolveFilename(f.configPath, location.Path)
	if err != nil {
		return nil, nil, err
	}

	authAccs := make([]*flowkit.Account, len(authorizers))
	for i, auth := range authorizers {
		authAccs[i] = createSigner(auth, service)
		if err != nil {
			return nil, nil, err
		}
	}

	accs, err := services.NewTransactionAccountRoles(service, service, authAccs)
	if err != nil {
		return nil, nil, err
	}

	return f.services.Transactions.Send(
		accs,
		&services.Script{
			Code:     code,
			Args:     args,
			Filename: codeFilename,
		},
		flow.DefaultTransactionGasLimit,
		config.DefaultEmulatorNetwork().Name,
	)
}

func (f *flowkitClient) GetAccount(address flow.Address) (*flow.Account, error) {
	return f.services.Accounts.Get(address)
}

func (f *flowkitClient) CreateAccount() (*clientAccount, error) {
	service, err := f.state.EmulatorServiceAccount()
	if err != nil {
		return nil, err
	}
	serviceKey, err := service.Key().PrivateKey()
	if err != nil {
		return nil, err
	}

	account, err := f.services.Accounts.Create(
		service,
		[]crypto.PublicKey{(*serviceKey).PublicKey()},
		[]int{flow.AccountKeyWeightThreshold},
		[]crypto.SignatureAlgorithm{crypto.ECDSA_P256},
		[]crypto.HashAlgorithm{crypto.SHA3_256},
		nil,
	)
	if err != nil {
		return nil, err
	}

	nextIndex := len(f.GetClientAccounts())
	if nextIndex > len(names) {
		return nil, fmt.Errorf(fmt.Sprintf("account limit of %d reached", len(names)))
	}

	clientAccount := &clientAccount{
		Account: account,
		Name:    names[nextIndex],
	}
	f.accounts = append(f.accounts, clientAccount)

	return clientAccount, nil
}

// Helpers
//

// createSigner creates a new flowkit account used for signing but using the key of the existing account.
func createSigner(address flow.Address, account *flowkit.Account) *flowkit.Account {
	signer := &flowkit.Account{}
	signer.SetAddress(address)
	signer.SetKey(account.Key())
	return signer
}

// resolveFilename helper converts the transaction file to a relative location to config file
// we will be replacing this logic once the FLIP is implemented
// https://github.com/onflow/flow/blob/master/flips/2022-03-23-contract-imports-syntax.md
func resolveFilename(configPath string, path string) (filename string, err error) {
	if filepath.Dir(configPath) != "." {
		filename, err = filepath.Rel(filepath.Dir(configPath), path)
		if err != nil {
			return "", err
		}
	}

	return filename, nil
}
