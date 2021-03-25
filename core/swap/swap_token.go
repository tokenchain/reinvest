package swap

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"reinvest/farm/mdex"
)

type LpToken struct {
	Client   *ethclient.Client
	Contract *mdex.SwapLpToken
}

func NewLpToken(address string, client *ethclient.Client) (*LpToken, error) {
	swapLpTokenContract, err := mdex.NewSwapLpToken(common.HexToAddress(address), client)
	if err != nil {
		return nil, err
	}
	return &LpToken{
		Client:   client,
		Contract: swapLpTokenContract,
	}, nil

}
func (c *LpToken) Pair() (string, string, error) {
	address0, err := c.Contract.Token0(&bind.CallOpts{})
	if err != nil {
		return "", "", err
	}
	address1, err := c.Contract.Token1(&bind.CallOpts{})
	if err != nil {
		return "", "", err
	}
	return address0.String(), address1.String(), nil
}

