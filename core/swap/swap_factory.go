package swap

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"reinvest/core/chain"
	"reinvest/farm/mdex"
	"reinvest/utils"
)

type SwapFactory struct {
	Client       *ethclient.Client
	SwapContract *mdex.Factory
	Chain        *chain.Chain
	Address      string
}

func NewSwapFactory(address string, client *ethclient.Client, chain *chain.Chain) (*SwapFactory, error) {
	swapFactory, err := mdex.NewFactory(common.HexToAddress(address), client)
	if err != nil {
		return nil, err
	}
	return &SwapFactory{
		Address:      address,
		Client:       client,
		SwapContract: swapFactory,
		Chain:        chain,
	}, nil

}

//获取期望兑换的值
func (c *SwapFactory) WishExchange(amountIn *big.Int, fromToken, toToken string) ([]*big.Int, error) {
	return c.SwapContract.GetAmountsOut(&bind.CallOpts{}, amountIn, []common.Address{
		common.HexToAddress(fromToken),
		common.HexToAddress(toToken),
	})
}

func (c *SwapFactory) Calc(amountB *big.Int, slippage float64) *big.Int {
	minAmount := new(big.Float).SetInt(amountB)

	min := new(big.Float).Mul(minAmount, big.NewFloat(1.00-slippage))
	a := big.NewInt(0)
	res, _ := min.Int(a)
	return res
}
func (c *SwapFactory) PairRatio(tokenA, tokenB string) (*big.Float, *big.Float, error) {
	reserveA, reserveB, err := c.GetReserves(tokenA, tokenB)
	if err != nil {
		return nil, nil, err
	}
	tokenAInfo, _ := c.Chain.TokenInfo(tokenA)
	tokenBInfo, _ := c.Chain.TokenInfo(tokenB)

	amountA := utils.ToWei("100", int(tokenAInfo.Decimals))
	amountB, err := c.SwapContract.Quote(&bind.CallOpts{}, amountA, reserveA, reserveB)
	if err != nil {
		return nil, nil, err
	}
	realAmountA := utils.ToDecimal(amountA, int(tokenAInfo.Decimals))
	realAmountB := utils.ToDecimal(amountB, int(tokenBInfo.Decimals))
	realAmountAFloat, _ := realAmountA.Float64()
	realAmountBFloat, _ := realAmountB.Float64()

	total := new(big.Float).Add(realAmountA.BigFloat(),realAmountB.BigFloat())
	log.Println("realAmountA:"+realAmountA.String())
	log.Println("realAmountB:"+amountB.String())

	log.Println("total:"+total.String())
	reserveAPairRatio := new(big.Float).Quo(big.NewFloat(realAmountAFloat), total)
	reserveBPairRatio := new(big.Float).Quo(big.NewFloat(realAmountBFloat), total)
	return  reserveBPairRatio,reserveAPairRatio, nil
}

func (c *SwapFactory) TokenBPairAmount(tokenA, tokenB string, amountA *big.Int) (*big.Int, error) {
	reserveA, reserveB, err := c.GetReserves(tokenA, tokenB)
	if err != nil {
		return nil, err
	}
	minB, err := c.SwapContract.Quote(&bind.CallOpts{}, amountA, reserveA, reserveB)
	if err != nil {
		return nil, nil
	}
	return minB, nil
}

func (c *SwapFactory) TokenAPairAmount(tokenA, tokenB string, amountB *big.Int) (*big.Int, error) {
	reserveA, reserveB, err := c.GetReserves(tokenA, tokenB)
	if err != nil {
		return nil, err
	}

	minB, err := c.SwapContract.Quote(&bind.CallOpts{}, amountB, reserveB, reserveA)
	if err != nil {
		return nil, nil
	}
	return minB, nil
}

func (c *SwapFactory) GetReserves(tokenA, tokenB string) (*big.Int, *big.Int, error) {
	reserves, err := c.SwapContract.GetReserves(&bind.CallOpts{}, common.HexToAddress(tokenA), common.HexToAddress(tokenB))
	if err != nil {
		return nil, nil, err
	}
	return reserves.ReserveA, reserves.ReserveB, nil
}
