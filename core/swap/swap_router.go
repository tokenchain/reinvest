package swap

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"io/ioutil"
	"math/big"
	"os"
	"reinvest/core/chain"
	"reinvest/farm/mdex"
	"reinvest/utils"
	"strings"
	"time"
)

type SwapRouter struct {
	Client       *ethclient.Client
	SwapContract *mdex.SwapRouter
	Chain        *chain.Chain
	Address      string
	Factory      string
}

func NewSwapRouter(address string, client *ethclient.Client, chain *chain.Chain) (*SwapRouter, error) {
	swapRouterContract, err := mdex.NewSwapRouter(common.HexToAddress(address), client)
	if err != nil {
		return nil, err
	}
	factory, _ := swapRouterContract.Factory(&bind.CallOpts{})
	return &SwapRouter{
		Factory:      factory.String(),
		Address:      address,
		Client:       client,
		SwapContract: swapRouterContract,
		Chain:        chain,
	}, nil

}
func (c *SwapRouter) SwapExactTokenTo(fromToken, toToken string, sendAmount, amountMin *big.Int) (*types.Transaction, error) {
	auth, err := c.Chain.CreateTx()
	if err != nil {
		return nil, err
	}
	wallet := c.Chain.WalletAddress()
	amountIn := sendAmount

	gasPrice, err := c.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	f, _ := os.Open(utils.BasePath("/abi/swap_router.abi"))
	defer f.Close()
	abiContent, _ := ioutil.ReadAll(f)
	ABI, err := abi.JSON(strings.NewReader(string(abiContent)))
	if err != nil {
		return nil, err
	}

	nonce, err := c.Client.PendingNonceAt(context.Background(), common.HexToAddress(wallet))
	if err != nil {
		return nil, err
	}
	t := time.Now()
	deadLine := t.Add(time.Hour * 24).Unix()
	toContract := common.HexToAddress(c.Address)
	//log.Println("amount  :" + amountIn.String())
	//log.Println("amount min  :" + amountMin.String())
	//log.Println("fromToken  :" + fromToken)
	//log.Println("toToken  :" + toToken)
	//log.Println("to  :" + wallet)
	//log.Println("deadLine :" + string(deadLine))

	txData, err := ABI.Pack("swapExactTokensForTokens", amountIn, amountMin, []common.Address{
		common.HexToAddress(fromToken),
		common.HexToAddress(toToken),
	}, common.HexToAddress(wallet), big.NewInt(deadLine))
	if err != nil {
		return nil, err
	}
	gas, err := c.Client.EstimateGas(context.Background(), ethereum.CallMsg{
		From:     common.HexToAddress(wallet),
		To:       &toContract,
		GasPrice: gasPrice,
		Value:    utils.ToWei(big.NewFloat(0.00), 18),
		Data:     txData,
	})

	if err != nil {
		return nil, nil
	}

	auth.GasPrice = gasPrice
	auth.From = common.HexToAddress(c.Chain.WalletAddress())
	auth.GasLimit = gas * 2
	auth.Context = context.Background()
	auth.Nonce = big.NewInt(int64(nonce))

	tx, err := c.SwapContract.SwapExactTokensForTokens(auth, amountIn, amountMin, []common.Address{
		common.HexToAddress(fromToken),
		common.HexToAddress(toToken),
	}, common.HexToAddress(wallet), big.NewInt(deadLine))
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (c *SwapRouter) AddLiquidity(tokenA, tokenB string, wishA, wishB, minA, minB *big.Int) (*types.Transaction, error) {
	auth, err := c.Chain.CreateTx()
	if err != nil {
		return nil, err
	}
	toContract := common.HexToAddress(c.Address)
	wallet := c.Chain.WalletAddress()
	gasPrice, err := c.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	f, _ := os.Open(utils.BasePath("/abi/swap_router.abi"))
	defer f.Close()
	abiContent, _ := ioutil.ReadAll(f)
	ABI, err := abi.JSON(strings.NewReader(string(abiContent)))
	if err != nil {
		return nil, err
		//log.Fatal(err)
	}
	nonce, err := c.Client.PendingNonceAt(context.Background(), common.HexToAddress(wallet))
	if err != nil {
		return nil, err
	}
	t := time.Now()
	deadLine := t.Add(time.Hour * 2).Unix()

	//log.Println("tokenA: " + tokenA)
	//log.Println("tokenB: " + tokenB)
	//
	//log.Println("tokenAWishAdd: " + tokenAWishAdd)
	//log.Println("tokenBWishAdd: " + tokenBWishAdd)
	//log.Println("minA: " + amountAMin)
	//log.Println("minB: " + amountBMin)

	txData, err := ABI.Pack(
		"addLiquidity",
		common.HexToAddress(tokenA),
		common.HexToAddress(tokenB),
		wishA,
		wishB,
		minA,
		minB,
		common.HexToAddress(wallet),
		big.NewInt(deadLine),
	)
	gas, err := c.Client.EstimateGas(context.Background(), ethereum.CallMsg{
		From:     common.HexToAddress(wallet),
		To:       &toContract,
		GasPrice: gasPrice,
		Value:    utils.ToWei(big.NewFloat(0.00), 18),
		Data:     txData,
	})

	if err != nil {
		return nil, err
	}

	auth.GasPrice = gasPrice
	auth.From = common.HexToAddress(c.Chain.WalletAddress())
	auth.GasLimit = gas * 2
	auth.Context = context.Background()
	auth.Nonce = big.NewInt(int64(nonce))

	tx, err := c.SwapContract.AddLiquidity(
		auth,
		common.HexToAddress(tokenA),
		common.HexToAddress(tokenB),
		wishA,
		wishB,
		minA,
		minB,
		common.HexToAddress(wallet),
		big.NewInt(deadLine),
	)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
