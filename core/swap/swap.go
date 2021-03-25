package swap

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"reinvest/core/chain"
	"reinvest/farm/mdex"
	"reinvest/utils"
)

type Swap struct {
	Client       *ethclient.Client
	SwapContract *mdex.SwapMining
	Chain        *chain.Chain
}

func NewSwap(client *ethclient.Client, SwapTokenAddress string, chain *chain.Chain) (*Swap, error) {
	swapContract, err := mdex.NewSwapMining(common.HexToAddress(SwapTokenAddress), client)
	if err != nil {
		return nil, err
	}
	return &Swap{Client: client, SwapContract: swapContract, Chain: chain}, nil
}
func (c *Swap) SwapProportion(rewardToken string, to string, amount *big.Int) (string, error) {
	toTokenInfo, err := c.Chain.TokenInfo(to)
	if err != nil {
		return "", err

	}
	proportion, err := c.SwapContract.GetQuantity(&bind.CallOpts{}, common.HexToAddress(to), amount, common.HexToAddress(rewardToken))
	if err != nil {
		return "", err
	}
	return utils.ToDecimal(proportion, int(toTokenInfo.Decimals)).String(), nil
}

func (c *Swap) SwapBalance(total string) (string, string) {

	totalAmount, _ := new(big.Float).SetString(total)
	remain := new(big.Float).Quo(totalAmount, big.NewFloat(2))
	return remain.String(), totalAmount.Sub(totalAmount, remain).String()

}
func (c *Swap) MinExchange(amount, proportion string) (string, error) {

	needExchange, _ := new(big.Float).SetString(amount)
	proportionBig, _ := new(big.Float).SetString(proportion)

	return new(big.Float).Mul(needExchange, proportionBig).String(), nil
}
