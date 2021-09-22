package conAbi

import (
	"bytes"
	"context"
	"contractDemo/transaction"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// HashLength is the expected length of the hash
	HashLength = 32
	// AddressLength is the expected length of the address
	AddressLength = 20
)

var (
	erc20ABI                           = transaction.ParseABIUnchecked(Erc20ABI)
	swapFactoryABI                     = transaction.ParseABIUnchecked(SimpleSwapFactoryABI)
	swapABI                            = transaction.ParseABIUnchecked(ERC20SimpleSwapABI)
	proxyABI                           = transaction.ParseABIUnchecked(ProxyAbi)
	oracleABI                          = transaction.ParseABIUnchecked(OracleAbi)
	errDecodeABI                       = errors.New("err decode abi")
	ErrTransactionReverted             = errors.New("err transaction reverted")
	ErrEventNotFound                   = errors.New("err event not found")
	ErrInvalidFactory                  = errors.New("invalid factory")
	ErrNotDeployedByFactory            = errors.New("not deployed by factory")
	SenderAdd                          = common.HexToAddress("0xA4E7663A031ca1f67eEa828E4795653504d38c6e")
	erc20Add                           = common.HexToAddress("0xD26c3d45a805a5f7809E27Bd18949d559e281900")
	tmpAdd                             = common.HexToAddress("0x9EAA021C41bf7644f68108913DCddd266caaa023")
	factoryAdd                         = common.HexToAddress("0x5E6802d9e7C8CD43BB7C96524fDD50FE8460B92c")
	simpleSwapDeployedEventType        = swapFactoryABI.Events["SimpleSwapDeployed"]
	currentDeployVersion        []byte = common.FromHex(FactoryDeployedBin)
	lock                        sync.Mutex
	DebugFlag                   = true
)

func GetBalance(ctx context.Context, address common.Address, backend transaction.Backend) (*big.Int, error) {
	callData, err := erc20ABI.Pack("balanceOf", address)
	if err != nil {
		return nil, err
	}
	if DebugFlag {
		fmt.Println("GetBalance: callData is ", callData)
	}

	output, err := Call(ctx, address, &transaction.TxRequest{
		To:   &erc20Add,
		Data: callData,
	}, backend)
	if err != nil {
		return nil, err
	}
	if DebugFlag {
		fmt.Println("GetBalance: output is ", output)
	}

	results, err := erc20ABI.Unpack("balanceOf", output)
	if err != nil {
		return nil, err
	}

	if len(results) != 1 {
		return nil, errDecodeABI
	}

	balance, ok := abi.ConvertType(results[0], new(big.Int)).(*big.Int)
	if !ok || balance == nil {
		return nil, errDecodeABI
	}
	return balance, nil
}

// Deploy deploys a new chequebook and returns once the transaction has been submitted.
func Deploy(ctx context.Context, issuer common.Address, defaultHardDepositTimeoutDuration *big.Int, nonce common.Hash, backend transaction.Backend, signer transaction.Signer, chainID *big.Int) (common.Hash, error) {
	callData, err := swapFactoryABI.Pack("deploySimpleSwap", issuer, big.NewInt(0).Set(defaultHardDepositTimeoutDuration), nonce)
	if err != nil {
		return common.Hash{}, err
	}

	gasPrice, err := backend.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatal(err)
	}

	request := &transaction.TxRequest{
		To:          &factoryAdd,
		Data:        callData,
		GasPrice:    gasPrice,
		GasLimit:    175000,
		Value:       big.NewInt(0),
		Description: "chequebook deployment",
	}

	txHash, err := Send(ctx, request, backend, issuer, signer, chainID)
	if err != nil {
		return common.Hash{}, err
	}

	return txHash, nil
}

// WaitDeployed waits for the deployment transaction to confirm and returns the chequebook address
func WaitDeployed(ctx context.Context, txHash common.Hash, backend transaction.Backend) (common.Address, error) {
	//简单一些，休眠20秒后再查询交易上链
	time.Sleep(time.Duration(20) * time.Second)
	receipt, err := backend.TransactionReceipt(ctx, txHash)
	if err != nil {
		return common.Address{}, fmt.Errorf("contract deployment failed: %w", err)
	}
	var event transaction.SimpleSwapDeployedEvent
	err = FindSingleEvent(&swapFactoryABI, receipt, factoryAdd, simpleSwapDeployedEventType, &event)
	if err != nil {
		return common.Address{}, fmt.Errorf("contract deployment failed: %w", err)
	}

	return event.ContractAddress, nil
}

func GetSwapBalance(ctx context.Context, address common.Address, backend transaction.Backend, swapAdd common.Address) (*big.Int, error) {
	callData, err := swapABI.Pack("balance")
	if err != nil {
		return nil, err
	}
	if DebugFlag {
		fmt.Println("GetSwapBalance: callData is ", callData)
	}

	output, err := Call(ctx, address, &transaction.TxRequest{
		To:   &swapAdd,
		Data: callData,
	}, backend)
	if err != nil {
		return nil, err
	}
	if DebugFlag {
		fmt.Println("GetSwapBalance: output is ", output)
	}

	results, err := swapABI.Unpack("balance", output)
	if err != nil {
		return nil, err
	}

	if len(results) != 1 {
		return nil, errDecodeABI
	}

	balance, ok := abi.ConvertType(results[0], new(big.Int)).(*big.Int)
	if !ok || balance == nil {
		return nil, errDecodeABI
	}
	return balance, nil
}

func PreWithdraw(ctx context.Context, fromaddress common.Address, backend transaction.Backend, swapAdd common.Address, signer transaction.Signer, chainID *big.Int) (common.Hash, error) {
	callData, err := swapABI.Pack("preWithdraw")
	if err != nil {
		return common.Hash{}, err
	}

	gasPrice, err := backend.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	request := &transaction.TxRequest{
		To:          &swapAdd,
		Data:        callData,
		GasPrice:    gasPrice,
		GasLimit:    175000,
		Value:       big.NewInt(0),
		Description: "pre withdraw",
	}

	txHash, err := Send(ctx, request, backend, fromaddress, signer, chainID)
	if err != nil {
		return common.Hash{}, err
	}

	return txHash, nil
}

func GetWithdrawTime(ctx context.Context, backend transaction.Backend, swapAdd common.Address) (*big.Int, error) {
	callData, err := swapABI.Pack("withdrawTime")
	if err != nil {
		return nil, err
	}

	output, err := Call(ctx, tmpAdd, &transaction.TxRequest{
		To:   &swapAdd,
		Data: callData,
	}, backend)
	if err != nil {
		return nil, err
	}
	if DebugFlag {
		fmt.Println("GetWithdrawTime: output is ", output)
	}

	results, err := swapABI.Unpack("withdrawTime", output)
	if err != nil {
		return nil, err
	}

	if len(results) != 1 {
		return nil, errDecodeABI
	}

	withdrawtime, ok := abi.ConvertType(results[0], new(big.Int)).(*big.Int)
	if !ok || withdrawtime == nil {
		return nil, errDecodeABI
	}

	return withdrawtime, nil
}

// VerifyBytecode checks that the factory is valid.
func VerifyFactoryBytecode(ctx context.Context, backend transaction.Backend) (err error) {
	code, err := backend.CodeAt(ctx, factoryAdd, nil)
	if err != nil {
		return err
	}
	if DebugFlag {
		fmt.Println("the deployed factory code is: ", code)
		fmt.Println("================================================================")
		fmt.Println("the version code is: ", currentDeployVersion)
	}
	if !bytes.Equal(code, currentDeployVersion) {
		return ErrInvalidFactory
	}

	return nil
}

func verifyChequebookAgainstFactory(ctx context.Context, chequebook common.Address, backend transaction.Backend) (bool, error) {
	callData, err := swapFactoryABI.Pack("deployedContracts", chequebook)
	if err != nil {
		return false, err
	}

	output, err := Call(ctx, tmpAdd, &transaction.TxRequest{
		To:   &factoryAdd,
		Data: callData,
	}, backend)
	if err != nil {
		return false, err
	}

	results, err := swapFactoryABI.Unpack("deployedContracts", output)
	if err != nil {
		return false, err
	}

	if len(results) != 1 {
		return false, errDecodeABI
	}

	deployed, ok := abi.ConvertType(results[0], new(bool)).(*bool)
	if !ok || deployed == nil {
		return false, errDecodeABI
	}
	if !*deployed {
		return false, nil
	}
	return true, nil
}

// VerifyChequebook checks that the supplied chequebook has been deployed by a supported factory.
func VerifyChequebook(ctx context.Context, chequebook common.Address, backend transaction.Backend) error {
	deployed, err := verifyChequebookAgainstFactory(ctx, chequebook, backend)
	if err != nil {
		return err
	}
	if deployed {
		return nil
	}

	return ErrNotDeployedByFactory
}

// ERC20Address returns the token for which this factory deploys chequebooks.
func ERC20Address(ctx context.Context, backend transaction.Backend) (common.Address, error) {
	callData, err := swapFactoryABI.Pack("ERC20Address")
	if err != nil {
		return common.Address{}, err
	}

	output, err := Call(ctx, tmpAdd, &transaction.TxRequest{
		To:   &factoryAdd,
		Data: callData,
	}, backend)
	if err != nil {
		return common.Address{}, err
	}

	results, err := swapFactoryABI.Unpack("ERC20Address", output)
	if err != nil {
		return common.Address{}, err
	}

	if len(results) != 1 {
		return common.Address{}, errDecodeABI
	}

	erc20Address, ok := abi.ConvertType(results[0], new(common.Address)).(*common.Address)
	if !ok || erc20Address == nil {
		return common.Address{}, errDecodeABI
	}
	return *erc20Address, nil
}

func GetMasterCopy(ctx context.Context, backend transaction.Backend, proxy common.Address) (common.Address, error) {

	callData, err := proxyABI.Pack("masterCopy")
	if err != nil {
		return common.Address{}, err
	}

	output, err := Call(ctx, tmpAdd, &transaction.TxRequest{
		To:   &proxy,
		Data: callData,
	}, backend)
	if err != nil {
		return common.Address{}, err
	}

	results, err := proxyABI.Unpack("masterCopy", output)
	if err != nil {
		return common.Address{}, err
	}

	if len(results) != 1 {
		return common.Address{}, errDecodeABI
	}

	master, ok := abi.ConvertType(results[0], new(common.Address)).(*common.Address)
	if !ok || master == nil {
		return common.Address{}, errDecodeABI
	}
	return *master, nil
}

//oracle
func GetOwner(ctx context.Context, backend transaction.Backend, oracle common.Address) (common.Address, error) {

	callData, err := oracleABI.Pack("owner")
	if err != nil {
		return common.Address{}, err
	}

	output, err := Call(ctx, tmpAdd, &transaction.TxRequest{
		To:   &oracle,
		Data: callData,
	}, backend)
	if err != nil {
		return common.Address{}, err
	}

	results, err := oracleABI.Unpack("owner", output)
	if err != nil {
		return common.Address{}, err
	}

	if len(results) != 1 {
		return common.Address{}, errDecodeABI
	}

	owner, ok := abi.ConvertType(results[0], new(common.Address)).(*common.Address)
	if !ok || owner == nil {
		return common.Address{}, errDecodeABI
	}
	return *owner, nil
}

func GetPrice(ctx context.Context, backend transaction.Backend, oracle common.Address) (*big.Int, error) {

	callData, err := oracleABI.Pack("price")
	if err != nil {
		return nil, err
	}

	output, err := Call(ctx, tmpAdd, &transaction.TxRequest{
		To:   &oracle,
		Data: callData,
	}, backend)
	if err != nil {
		return nil, err
	}

	results, err := oracleABI.Unpack("price", output)
	if err != nil {
		return nil, err
	}

	if len(results) != 1 {
		return nil, errDecodeABI
	}

	price, ok := abi.ConvertType(results[0], new(big.Int)).(*big.Int)
	if !ok || price == nil {
		return nil, errDecodeABI
	}
	return price, nil
}

func Call(ctx context.Context, address common.Address, request *transaction.TxRequest, backend transaction.Backend) ([]byte, error) {
	msg := ethereum.CallMsg{
		From:     address,
		To:       request.To,
		Data:     request.Data,
		GasPrice: request.GasPrice,
		Gas:      request.GasLimit,
		Value:    request.Value,
	}
	data, err := backend.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Send creates and signs a transaction based on the request and sends it.
func Send(ctx context.Context, request *transaction.TxRequest, backend transaction.Backend, sender common.Address, signer transaction.Signer, chainID *big.Int) (txHash common.Hash, err error) {
	lock.Lock()
	defer lock.Unlock()

	nonce, err := backend.PendingNonceAt(context.Background(), sender)
	if err != nil {
		fmt.Println("Send err 1")
		return common.Hash{}, err
	}

	tx, err := prepareTransaction(ctx, request, sender, backend, nonce)
	if err != nil {
		return common.Hash{}, err
	}

	signedTx, err := signer.SignTx(tx, chainID)
	if err != nil {
		return common.Hash{}, err
	}

	err = backend.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Hash{}, err
	}

	txHash = signedTx.Hash()

	//wait for 20 secoonds
	time.Sleep(time.Duration(20 * time.Second))

	return signedTx.Hash(), nil
}

// prepareTransaction creates a signable transaction based on a request.
func prepareTransaction(ctx context.Context, request *transaction.TxRequest, from common.Address, backend transaction.Backend, nonce uint64) (tx *types.Transaction, err error) {
	var gasLimit uint64
	if request.GasLimit == 0 {
		gasLimit, err = backend.EstimateGas(ctx, ethereum.CallMsg{
			From: from,
			To:   request.To,
			Data: request.Data,
		})
		if err != nil {
			return nil, err
		}

		gasLimit += gasLimit / 5 // add 20% on top

	} else {
		gasLimit = request.GasLimit
	}

	var gasPrice *big.Int
	if request.GasPrice == nil {
		gasPrice, err = backend.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		gasPrice = request.GasPrice
	}

	if request.To != nil {
		return types.NewTransaction(
			nonce,
			*request.To,
			request.Value,
			gasLimit,
			gasPrice,
			request.Data,
		), nil
	}

	return types.NewContractCreation(
		nonce,
		request.Value,
		gasLimit,
		gasPrice,
		request.Data,
	), nil
}

// ParseEvent will parse the specified abi event from the given log
func ParseEvent(a *abi.ABI, eventName string, c interface{}, e types.Log) error {
	if len(e.Topics) == 0 {
		return errors.New("err no topic")
	}
	if len(e.Data) > 0 {
		if err := a.UnpackIntoInterface(c, eventName, e.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range a.Events[eventName].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(c, indexed, e.Topics[1:])
}

// FindSingleEvent will find the first event of the given kind.
func FindSingleEvent(abi *abi.ABI, receipt *types.Receipt, contractAddress common.Address, event abi.Event, out interface{}) error {
	if receipt.Status != 1 {
		return ErrTransactionReverted
	}
	for _, log := range receipt.Logs {
		if log.Address != contractAddress {
			continue
		}
		if len(log.Topics) == 0 {
			continue
		}
		if log.Topics[0] != event.ID {
			continue
		}

		return ParseEvent(abi, event.Name, out, *log)
	}
	return ErrEventNotFound
}
