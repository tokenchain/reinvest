package main

import (
	"bufio"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"reinvest/core/chain"
	"reinvest/core/swap"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	"log"
	"math/big"
	"reinvest/utils"
)

var blue = color.New(color.FgHiBlue).SprintFunc()
var green = color.New(color.FgHiGreen).SprintFunc()
var red = color.New(color.FgHiRed).SprintFunc()

func main() {

	var router = "0xED7d5F38C79115ca12fe6C0041abb22F0A06C300"
	var rewardToken = "0x25d2e80cb6b86881fd7e07dd263fb79f4abe033c"
	var farmAddress = "0xFB03e11D93632D97a8981158A632Dd5986F5E909"
	wallet := ""
	client, err := ethclient.Dial("https://http-mainnet-node.huobichain.com")
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Pool Id: ")
	poolIDString, _ := reader.ReadString('\n')
	poolIDString = strings.Replace(poolIDString, "\n", "", -1)
	if poolIDString == "" {
		fmt.Println(red("Pool Id Empty "))
		return
	}
	poolID, err := strconv.Atoi(poolIDString)
	if err != nil {
		fmt.Println(red("Invaild Pool Id"))
		return
	}
	chain := &chain.Chain{
		Client: client,
	}
	count := 0
	for {
		if count >= 4 {
			fmt.Print("Too many attempts Press Any Key to Exit...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
			return
		}
		fmt.Print("Enter Private Key: ")

		bytePrivateKey, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println(red("Invaild Private Key "))
			count++
			continue
		}
		privateKey := string(bytePrivateKey)
		if err != nil {
			fmt.Println(red("Invaild Private Key "))
			count++

			continue
		}
		walletAddress, err := chain.CheckPrivateKey(privateKey)
		if err != nil {
			fmt.Println(red("Invaild Private Key "))
			count++

			continue
		}
		if walletAddress == "" {
			fmt.Println(red("Invaild Private Key "))
			count++

			continue
		}
		wallet = walletAddress
		chain.PrivateKey = privateKey
		break

	}

	//addLiquidity(big.NewInt(275651401900), big.NewInt(8156087518776), "0x25d2e80cb6b86881fd7e07dd263fb79f4abe033c", "0xa71edc38d189767582c38a3145b5873052c3e47a", client, chain)
	farmInfo, err := chain.GetPoolInfo(farmAddress, poolID)
	if err != nil {
		log.Printf("Get Farm  err  %w ", red(err))

	}

	writer := uilive.New()
	writer.Start()
	LpToken, err := swap.NewLpToken(farmInfo.LpToken, client)
	if err != nil {
		log.Printf("Get LpToken Info Err %w  \n", err)
	}
	address0, address1, err := LpToken.Pair()
	if err != nil {
		log.Printf("Get Token Pair Info Err %w  \n", err)
	}
	swapRouter, err := swap.NewSwapRouter(router, client, chain)
	if err != nil {
		log.Printf("Get Router  Err %w  \n", err)

	}
	factory, err := swap.NewSwapFactory(swapRouter.Factory, client, chain)
	if err != nil {
		log.Printf("Get Factory  Err %w  \n", err)
	}
	sortRes, err := factory.SwapContract.SortTokens(&bind.CallOpts{}, common.HexToAddress(address0), common.HexToAddress(address1))
	if err != nil {
		log.Printf("Sort Tokens  Err %w  \n", err)
	} else {
		address0 = sortRes.Token0.String()
		address1 = sortRes.Token1.String()
	}

	rewardTokenInfo, err := chain.TokenInfo(rewardToken)
	if err != nil {
		log.Printf("Get Reward Token Info   Err  %s \n", red(err))
	}
	tokenA := address0
	tokenB := address1
	tokenAInfo, err := chain.GetMyTokenInfo(tokenA)
	if err != nil {
		log.Printf("Get My Token A Info  Err  %s \n", red(err))
	}
	tokenBInfo, err := chain.GetMyTokenInfo(tokenB)
	if err != nil {
		log.Printf("Get My Token B Info  Err  %s \n", red(err))
	}
	fmt.Println("")
	fmt.Println("Start...")
	time.Sleep(time.Millisecond * 1000)
	for {
		userInfo, err := chain.GetFarmUserInfo(farmAddress, wallet, poolID)
		lpToken, err := chain.TokenInfo(farmInfo.LpToken)
		if err != nil {
			log.Printf("Get Farm  err  %w ", red(err))
		}
		fmt.Printf("Current have %s %s in pool  \n", green(utils.ToDecimal(userInfo.Amount, int(lpToken.Decimals))), green(lpToken.Symbol))
		if err != nil {
			log.Printf("Get Farm UserInfo err  %w ", red(err))
		}
		myLpTokenInfo, err := chain.GetMyTokenInfo(farmInfo.LpToken)
		if err != nil {
			log.Printf(red("Get My Token Info   Err  %s \n", red(err)))
		}
		pendingReward, err := chain.Pending(farmAddress, wallet, poolID)
		if err != nil {
			log.Printf("Get Pending Rewards  Err  %s \n", red(err))
		}
		fmt.Printf("Wallet has %s %s %s \n", green(utils.ToDecimal(myLpTokenInfo.Balance, int(myLpTokenInfo.Decimals))), green(myLpTokenInfo.Symbol), blue("- "+myLpTokenInfo.Name))
		fmt.Printf("Pending Reward in Pool  %s MDX \n", green(utils.ToDecimal(pendingReward.Amount, int(rewardTokenInfo.Decimals))))
		if pendingReward.Amount.Cmp(big.NewInt(0)) >= 1 {
			realPendingRewardAmount := big.NewInt(0)
			tx, err := chain.Deposit(farmAddress, big.NewInt(0), poolID)
			if err != nil {
				log.Println(red("Harvest error: " + err.Error()))
				continue
			}
			fmt.Println(blue(fmt.Sprintf("Harvest Reward Tx: %s ", tx.Hash().String())))
			txStatus, _tx := chain.WaitForBlockCompletation(tx.Hash().String())
			if txStatus != 1 {
				fmt.Println(red("Harvest Err Tx :" + tx.Hash().String()))
			} else {
				fmt.Println(green("Harvest Success Tx:" + tx.Hash().String()))
			}
			if txStatus != 1 {
				continue
			}
			sendRewardAmountToWallet, err := chain.GetTxAmount(rewardToken, farmAddress, wallet, _tx)
			if err != nil {
				log.Println(red("Search Real Reward Amount To Wallet Error: " + err.Error()))
				continue
			}
			realPendingRewardAmount = sendRewardAmountToWallet
			fmt.Println(green(utils.ToDecimal(realPendingRewardAmount, int(rewardTokenInfo.Decimals)).String()+" "+rewardTokenInfo.Symbol) + " -> " + wallet)
			avg := new(big.Int).Div(realPendingRewardAmount, big.NewInt(2))
			realTokenASwapAmount := big.NewInt(0)
			if strings.ToLower(tokenA) != strings.ToLower(rewardToken) {
				swapTx, err := Swap(avg, rewardToken, tokenA, router, client, chain)
				if err != nil {
					log.Println(red("swap error " + err.Error()))
					continue
				}
				fmt.Println(blue(fmt.Sprintf("MDX -> %s Tx: %s ", tokenAInfo.Symbol, swapTx.Hash().String())))
				swapTxStatus, _tx := chain.WaitForBlockCompletation(swapTx.Hash().String())
				if swapTxStatus == 1 {
					log.Println(green("Swap MDX -> " + tokenAInfo.Symbol + " Success Tx: " + swapTx.Hash().String()))
				} else {
					log.Println(red("Swap  Token A Error Tx: " + swapTx.Hash().String()))
				}
				if swapTxStatus != 1 {
					continue
				}

				sendAmountToWallet, err := chain.GetTxAmount(tokenA, "", wallet, _tx)
				if err != nil {
					log.Println(red("Search Real Amount Token A To Wallet Error: " + err.Error()))
					continue
				}
				realTokenASwapAmount = sendAmountToWallet
				fmt.Println(green(utils.ToDecimal(realTokenASwapAmount, int(tokenAInfo.Decimals)).String()+" "+tokenAInfo.Symbol) + " -> " + wallet)
			} else {
				realTokenASwapAmount = avg
			}
			time.Sleep(time.Second * 2)
			realTokenBSwapAmount := big.NewInt(0)
			if strings.ToLower(tokenB) != strings.ToLower(rewardToken) {
				swapTx, err := Swap(avg, rewardToken, tokenB, router, client, chain)
				if err != nil {
					log.Println(red("swap error " + err.Error()))
					continue
				}
				fmt.Println(blue(fmt.Sprintf("Swap MDX -> %s Tx: %s ", tokenBInfo.Symbol, green(swapTx.Hash().String()))))
				swapTxStatus, _tx := chain.WaitForBlockCompletation(swapTx.Hash().String())
				if swapTxStatus == 1 {
					log.Println(green("Swap Token B Success Tx: " + swapTx.Hash().String()))
				} else {
					log.Println(red("Swap Token B Error Tx: " + swapTx.Hash().String()))
				}
				if swapTxStatus != 1 {
					continue
				}
				sendAmountToWallet, err := chain.GetTxAmount(tokenB, "", wallet, _tx)
				if err != nil {
					log.Println(red("Search Real Amount Token B To Wallet Error: " + err.Error()))
					continue
				}
				realTokenBSwapAmount = sendAmountToWallet
				fmt.Println(green(utils.ToDecimal(realTokenBSwapAmount, int(tokenBInfo.Decimals)).String()+" "+tokenBInfo.Symbol) + " -> " + wallet)

			} else {
				realTokenBSwapAmount = avg
			}
			time.Sleep(time.Second * 2)
			//fmt.Printf("total has %s %s swap %s %s ->  %s \n", rewardTokenInfo.Balance, rewardTokenInfo.Symbol, utils.ToDecimal(avg, int(rewardTokenInfo.Decimals)), rewardTokenInfo.Symbol, toTokenInfo.Symbol)

			addLiquidityTx, err := addLiquidity(realTokenASwapAmount, realTokenBSwapAmount, router, tokenA, tokenB, client, chain)
			if err != nil {
				log.Println(red("Rreinvest Error: ", err.Error()))
				continue
			}
			fmt.Println(blue(fmt.Sprintf("addLiquidity Tx : %s \n", addLiquidityTx.Hash().String())))
			addLiquidityTxStatus, _ := chain.WaitForBlockCompletation(addLiquidityTx.Hash().String())
			if addLiquidityTxStatus == 1 {
				log.Println(green("addLiquidity success txh: " + addLiquidityTx.Hash().String()))
			} else {
				log.Println(red("addLiquidity error txh: " + addLiquidityTx.Hash().String()))
			}
			if addLiquidityTxStatus != 1 {
				continue
			}

			lpTokenApprove, err := chain.Approve(farmInfo.LpToken, farmAddress, realTokenBSwapAmount)
			if err != nil {
				fmt.Println("Approve LP Token Error: " + red(err))
				continue
			}
			if !lpTokenApprove {
				fmt.Println(red("Approve LP Token  Fail"))
				continue
			}
			LpTokenBalance, err := chain.GetMyTokenInfo(farmInfo.LpToken)
			if err != nil {
				log.Printf("Get My LP Token Info  Err  %s \n", red(err))
				continue
			}
			fmt.Println(blue(fmt.Sprintf("%s %s -> Farm ", utils.ToDecimal(LpTokenBalance.Balance, int(LpTokenBalance.Decimals)).String(), LpTokenBalance.Symbol)))
			depTx, err := chain.Deposit(farmAddress, LpTokenBalance.Balance, poolID)
			if err != nil {
				fmt.Println(red("Deposit Error: " + err.Error()))
				continue
			}
			depTxStatus, _ := chain.WaitForBlockCompletation(depTx.Hash().String())
			if depTxStatus != 1 {
				log.Println(red("Deposit Err Txh :" + depTx.Hash().String()))
				continue

			} else {
				log.Println(green("Deposit Success Txh :" + depTx.Hash().String()))
			}
		}

		time.Sleep(time.Minute * 5)
	}
	writer.Stop()

}

//配对
func addLiquidity(wishA *big.Int, wishB *big.Int, router string, tokenA, tokenB string, client *ethclient.Client, c *chain.Chain) (*types.Transaction, error) {
	if wishA.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("Error Token A Wish Amount ")
	}
	if wishB.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("Error Token B Wish Amount ")
	}
	approved, err := c.Approve(tokenA, router, wishA)
	if err != nil {
		return nil, fmt.Errorf("Approve Token A Error : %w", err)
	}
	if !approved {
		return nil, fmt.Errorf("Approve Token A Fail")
	}
	//fmt.Println(green("Has Approved To :" + router + " for Token A :" + tokenA))
	tokenBApproved, err := c.Approve(tokenB, router, wishB)
	if err != nil {
		return nil, fmt.Errorf("Approve Token B Error : %w", err)
	}
	if !tokenBApproved {
		return nil, fmt.Errorf("Approve Token B Fail")
	}
	swapRouter, err := swap.NewSwapRouter(router, client, c)
	if err != nil {
		return nil, err
	}
	factory, err := swap.NewSwapFactory(swapRouter.Factory, client, c)
	//needSendTo := harvestBalance
	currentCanPairTokenB, err := factory.TokenBPairAmount(tokenA, tokenB, wishA)
	if err != nil {
		return nil, err
	}
	minB := factory.Calc(currentCanPairTokenB, 0.005)
	minA := factory.Calc(wishA, 0.005)
	//fmt.Println(green("Has Approved To: " + router + " for Token B: " + tokenB))
	return swapRouter.AddLiquidity(tokenA, tokenB, wishA, wishB, minA, minB)
}

func Swap(rewardAmount *big.Int, from, to string, router string, client *ethclient.Client, c *chain.Chain) (*types.Transaction, error) {
	approved, err := c.Approve(from, router, rewardAmount)
	if err != nil {
		return nil, fmt.Errorf("Approve Swap Token Error : %w", err)
	}
	if !approved {
		return nil, fmt.Errorf("Approve Swap Token Fail")
	}
	//fmt.Println(green("Has Approved To :" + router + " for :" + from))
	swapRouter, err := swap.NewSwapRouter(router, client, c)
	if err != nil {
		return nil, err
	}
	factory, err := swap.NewSwapFactory(swapRouter.Factory, client, c)

	wishAmount, err := factory.WishExchange(rewardAmount, from, to)
	minExchange := factory.Calc(wishAmount[1], 0.005)

	tx, err := swapRouter.SwapExactTokenTo(from, to, rewardAmount, minExchange)
	if err != nil {
		return nil, err
	}
	return tx, nil

}
