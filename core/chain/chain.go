package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"reinvest/farm/mdex"
	"reinvest/utils"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"reinvest/token"
)

type Chain struct {
	Client     *ethclient.Client
	Wallet     string
	PrivateKey string
}

type MyTokenInfo struct {
	Balance  *big.Int
	Name     string
	Decimals uint8
	Symbol   string
}
type FarmUserInfo struct {
	Amount           *big.Int
	RewardDebt       string
	MultLpRewardDebt string
}
type PoolInfo struct {
	FarmContract      *mdex.MdexFarm
	LpToken           string
	AllocPoint        string
	LastRewardBlock   string
	AccMdxPerShare    string
	AccMultLpPerShare string
	TotalAmount       string
}

func (c *Chain) CheckPrivateKey(key string) (string, error) {
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		return "", err
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		//log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return "", errors.New("Invaild Private Key")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA).String(), nil
}
func (c *Chain) WalletAddress() string {
	privateKey, err := crypto.HexToECDSA(c.PrivateKey)
	if err != nil {
		return ""
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		//log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return ""
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA).String()
}

func (c *Chain) CreateTx() (*bind.TransactOpts, error) {

	privateKey, err := crypto.HexToECDSA(c.PrivateKey)
	if err != nil {
		return nil, err
	}

	chainID, err := c.Client.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}
	return bind.NewKeyedTransactorWithChainID(privateKey, chainID)
}

func (c *Chain) Deposit(farmAddress string, amount *big.Int, pool int) (*types.Transaction, error) {

	if !utils.IsValidAddress(farmAddress) {
		return nil, errors.New("Farm Address Is InValid!")
	}
	poolInfo, err := c.GetPoolInfo(farmAddress, pool)
	if err != nil {
		return nil, fmt.Errorf("Get Pool Info Error : %w", err)
	}
	wallet := c.WalletAddress()

	gasPrice, err := c.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Get Gas Price Error %w", err)
	}

	nonce, err := c.Client.PendingNonceAt(context.Background(), common.HexToAddress(wallet))
	if err != nil {
		return nil, fmt.Errorf("Get Nonce Error %w", err)
	}
	auth, err := c.CreateTx()
	if err != nil {
		return nil, fmt.Errorf("Get Create Transaction Error %w", err)
	}
	auth.GasPrice = gasPrice
	auth.From = common.HexToAddress(wallet)
	auth.GasLimit = uint64(300000)
	auth.Context = context.Background()
	auth.Nonce = big.NewInt(int64(nonce))
	tx, err := poolInfo.FarmContract.Deposit(auth, big.NewInt(int64(pool)), amount)
	if err != nil {
		return nil, fmt.Errorf("Deposit Errx %w", err)
	}
	return tx, nil

}
func (c *Chain) CheckTransactionReceipt(_txHash string) int {
	txHash := common.HexToHash(_txHash)
	tx, err := c.Client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return -1
	}
	return (int(tx.Status))
}
func (c *Chain) WaitForBlockCompletation(hash string) (int, *types.Receipt) {
	ctx, chancel := context.WithTimeout(context.Background(), time.Second*60)
	defer chancel()
	transaction := make(chan *types.Receipt)
	go func(context context.Context, client *ethclient.Client) {
		for {
			statusCode := -1
			txHash := common.HexToHash(hash)
			tx, err := client.TransactionReceipt(ctx, txHash)
			//tx.BlockNumber.String()
			if err == nil {
				statusCode = int(tx.Status)
				transaction <- tx
				return
			} else {
				statusCode = -1
			}
			select {
			case <-ctx.Done():
				if statusCode == -1 {
					transaction <- nil
				} else {
					transaction <- tx
				}
				break
			default:
				_ = 1
			}
			time.Sleep(time.Second * 2)
		}
	}(ctx, c.Client)
	select {
	case tx := <-transaction:
		if tx != nil {
			return int(tx.Status), tx
		}
		return -1, nil
	}

}

func (c *Chain) GetPoolInfo(farmAddress string, pool int) (*PoolInfo, error) {
	if !utils.IsValidAddress(farmAddress) {
		return nil, errors.New("Farm Address Is InValid!")
	}
	farm, err := mdex.NewMdexFarm(common.HexToAddress(farmAddress), c.Client)

	if err != nil {
		return nil, fmt.Errorf("Get Farm Error : %w", err)
	}
	poolInfo, err := farm.PoolInfo(&bind.CallOpts{}, new(big.Int).SetInt64(int64(pool)))
	if err != nil {
		return nil, fmt.Errorf("Get Pool Info Error : %w", err)
	}
	return &PoolInfo{

		FarmContract:      farm,
		LpToken:           poolInfo.LpToken.String(),
		AllocPoint:        poolInfo.AllocPoint.String(),
		LastRewardBlock:   poolInfo.LastRewardBlock.String(),
		AccMdxPerShare:    poolInfo.AccMdxPerShare.String(),
		AccMultLpPerShare: poolInfo.AccMultLpPerShare.String(),
		TotalAmount:       poolInfo.TotalAmount.String(),
	}, nil
}

type PendingReward struct {
	Amount      *big.Int
	TokenAmount *big.Int
}

func (c *Chain) Pending(farmAddress string, wallet string, pool int) (*PendingReward, error) {
	if !utils.IsValidAddress(farmAddress) {
		return nil, errors.New("Farm Address Is InValid!")
	}
	if !utils.IsValidAddress(wallet) {
		return nil, errors.New("Wallet Address Is InValid!")
	}
	//质押池信息
	poolInfo, err := c.GetPoolInfo(farmAddress, pool)
	if err != nil {
		return nil, fmt.Errorf("Get Pool Info Error : %v", err)
	}
	amount, tokenAmount, err := poolInfo.FarmContract.Pending(&bind.CallOpts{}, new(big.Int).SetInt64(int64(pool)), common.HexToAddress(wallet))
	if err != nil {
		return nil, fmt.Errorf("Get Pool Pending Reward  Error : %v", err)

	}

	return &PendingReward{
		Amount:      amount,
		TokenAmount: tokenAmount,
	}, nil

}

//获取我的信息
func (c *Chain) GetFarmUserInfo(farmAddress string, wallet string, pool int) (*FarmUserInfo, error) {
	if !utils.IsValidAddress(farmAddress) {
		return nil, errors.New("Farm Address Is InValid!")
	}
	if !utils.IsValidAddress(wallet) {
		return nil, errors.New("Wallet Address Is InValid!")
	}
	//质押池信息
	poolInfo, err := c.GetPoolInfo(farmAddress, pool)
	if err != nil {
		return nil, fmt.Errorf("Get Pool Info  Error : %v", err)
	}

	userInfo, err := poolInfo.FarmContract.UserInfo(&bind.CallOpts{}, new(big.Int).SetInt64(int64(pool)), common.HexToAddress(wallet))
	if err != nil {
		return nil, fmt.Errorf("Get User Info  Error : %v", err)
	}

	return &FarmUserInfo{
		Amount:           userInfo.Amount,
		RewardDebt:       userInfo.RewardDebt.String(),
		MultLpRewardDebt: userInfo.MultLpRewardDebt.String(),
	}, nil
}

type Token struct {
	TokenContract *token.Hrc20
	Name          string
	Decimals      uint8
	Symbol        string
}

func (c *Chain) TokenInfo(tokenAddress string) (*Token, error) {
	if !utils.IsValidAddress(tokenAddress) {
		return nil, errors.New("Token Address Is InValid!")
	}
	tokenContract, err := token.NewHrc20(common.HexToAddress(tokenAddress), c.Client)
	if err != nil {

		return nil, errors.New("Get Token Error")

	}
	decimals, err := tokenContract.Decimals(&bind.CallOpts{})
	if err != nil {
		return nil, errors.New("Get Decimals Error")
	}
	tokenName, err := tokenContract.Name(&bind.CallOpts{})
	if err != nil {
		return nil, errors.New("Get Token Name Error")
	}
	symbol, err := tokenContract.Symbol(&bind.CallOpts{})
	if err != nil {
		return nil, errors.New("Get Token Name Error")
	}
	return &Token{
		TokenContract: tokenContract,
		Name:          tokenName,
		Decimals:      decimals,
		Symbol:        symbol,
	}, nil
}

func (c *Chain) GetMyTokenInfo(tokenAddress string) (*MyTokenInfo, error) {
	walletAddress := c.WalletAddress()
	if !utils.IsValidAddress(tokenAddress) {
		return nil, errors.New("Token Address Is InValid!")
	}
	if !utils.IsValidAddress(walletAddress) {
		return nil, errors.New("Wallet Address Is InValid!")
	}
	token, err := c.TokenInfo(tokenAddress)
	if err != nil {

		return nil, errors.New("Get Token Error")
	}
	balance, err := token.TokenContract.BalanceOf(&bind.CallOpts{}, common.HexToAddress(walletAddress))
	if err != nil {
		return nil, errors.New("Get Balance Error")
	}

	return &MyTokenInfo{
		Balance:  balance,
		Name:     token.Name,
		Decimals: token.Decimals,
		Symbol:   token.Symbol,
	}, nil
}

func (c *Chain) Approve(tokenAddress string, contractAddress string, amount *big.Int) (bool, error) {
	wallet := c.WalletAddress()
	hrc20, err := token.NewHrc20(common.HexToAddress(tokenAddress), c.Client)
	if err != nil {
		return false, err
	}
	allowance, err := hrc20.Allowance(&bind.CallOpts{}, common.HexToAddress(wallet), common.HexToAddress(contractAddress))
	if err != nil {
		return false, err
	}
	if allowance.Cmp(amount) >= 1 {

		return true, nil
	}
	gasPrice, err := c.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return false, err
	}

	ABI, err := abi.JSON(strings.NewReader(token.Hrc20ABI))
	if err != nil {
		return false, err
	}
	unlimit := new(big.Int).Exp(big.NewInt(2), big.NewInt(250), nil)
	txData, err := ABI.Pack("approve", common.HexToAddress(contractAddress), unlimit.Sub(unlimit, big.NewInt(1)))
	if err != nil {
		return false, err
	}
	toContract := common.HexToAddress(tokenAddress)
	gas, err := c.Client.EstimateGas(context.Background(), ethereum.CallMsg{
		From:     common.HexToAddress(c.WalletAddress()),
		To:       &toContract,
		GasPrice: gasPrice,
		Value:    utils.ToWei(big.NewFloat(0.00), 18),
		Data:     txData,
	})
	nonce, err := c.Client.PendingNonceAt(context.Background(), common.HexToAddress(wallet))
	if err != nil {
		return false, fmt.Errorf("Get Nonce Error %w", err)
	}
	auth, err := c.CreateTx()
	if err != nil {
		return false, err
	}
	auth.GasPrice = gasPrice
	auth.From = common.HexToAddress(c.WalletAddress())
	auth.GasLimit = gas * 2
	auth.Context = context.Background()
	auth.Nonce = big.NewInt(int64(nonce))

	tx, err := hrc20.Approve(auth, common.HexToAddress(contractAddress), unlimit.Sub(unlimit, big.NewInt(1)))
	if err != nil {
		return false, err
	}
	fmt.Println("Approve Tx: " + tx.Hash().String())
	txStatus, _ := c.WaitForBlockCompletation(tx.Hash().String())
	if txStatus == 1 {
		return true, nil
	}
	return false, errors.New("Approve Faild")
}

func (c *Chain) GetTxAmount(tokenAddress, from, to string, tx *types.Receipt) (*big.Int, error) {
	hrc20, err := token.NewHrc20(common.HexToAddress(tokenAddress), c.Client)
	if err != nil {
		return nil, err
	}
	end := uint64(tx.BlockNumber.Int64())
	var fromCommon []common.Address
	if from != "" {
		fromCommon = []common.Address{common.HexToAddress(from)}
	}

	filter, err := hrc20.FilterTransfer(&bind.FilterOpts{Start: uint64(tx.BlockNumber.Int64()), End: &end}, fromCommon, []common.Address{common.HexToAddress(to)})
	if err != nil {
		return nil, err
	}
	for {
		if filter.Next() {

			if tx.TxHash.String() == filter.Event.Raw.TxHash.String() {
				return filter.Event.Value, nil
			}
		} else {
			return nil, errors.New("not found")
		}
	}
	return nil, errors.New("not found")
}
