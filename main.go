package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/manifoldco/promptui"
	"os"
	"reinvest/core/chain"
	"reinvest/core/swap"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fatih/color"
	"log"
	"math/big"
	"reinvest/utils"
)

var blue func(a ...interface{}) string
var green func(a ...interface{}) string
var red func(a ...interface{}) string
var wallet string
var reinvestInterval int

func main() {
	defer func() {
		// 发生宕机时，获取panic传递的上下文并打印
		err := recover()
		switch err.(type) {
		case runtime.Error: // 运行时错误
			fmt.Println("runtime error:", err)
		default: // 非运行时错误
			fmt.Println("error:", err)
		}
	}()

	sysType := runtime.GOOS

	if sysType == "linux" || sysType == "darwin" {
		blue = color.New(color.FgHiBlue).SprintFunc()
		green = color.New(color.FgHiGreen).SprintFunc()
		red = color.New(color.FgHiRed).SprintFunc()

		// LINUX系统
	} else {
		blue = func(a ...interface{}) string {
			return fmt.Sprint(a...)
		}
		green = func(a ...interface{}) string {
			return fmt.Sprint(a...)
		}
		red = func(a ...interface{}) string {
			return fmt.Sprint(a...)
		}
	}

	var router = "0xED7d5F38C79115ca12fe6C0041abb22F0A06C300"
	var rewardToken = "0x25d2e80cb6b86881fd7e07dd263fb79f4abe033c"
	var farmAddress = "0xFB03e11D93632D97a8981158A632Dd5986F5E909"
	client, err := ethclient.Dial("https://http-mainnet-node.huobichain.com")
	if err != nil {
		log.Fatal(err)
	}
	poolIdValidate := func(input string) error {
		_, err := strconv.Atoi(input)
		if err != nil {
			return errors.New("Invalid Pool ID")
		}
		return nil
	}
	prompt := promptui.Prompt{
		Label:    "Pool ID",
		Validate: poolIdValidate,
	}

	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		pause()
		return
	}
	poolID, err := strconv.Atoi(result)
	if err != nil {
		fmt.Println("Invalid Pool ID")
		pause()
		return
	}
	chain := &chain.Chain{
		Client: client,
	}

	privateValidate := func(input string) error {
		input = strings.Replace(input, " ", "", -1)
		input = strings.Replace(input, "\n", "", -1)
		wallet, err := chain.CheckPrivateKey(input)
		if err != nil {
			return fmt.Errorf("Invaild Private Key ")

		}
		if wallet == "" {
			return errors.New("Invaild Private Key ")
		}
		return nil
	}
	pk := promptui.Prompt{
		Label:    "Private Key",
		Validate: privateValidate,
		Mask:     '*',
	}
	privateKey, err := pk.Run()
	if err != nil {
		fmt.Println(red(err.Error()))
		pause()
		return
	}
	minutesValidate := func(input string) error {
		_, err := strconv.Atoi(input)
		if err != nil {
			return errors.New("Invalid Reinvest Time Interval")
		}
		return nil
	}
	between := promptui.Prompt{
		Label:    "Reinvest  Interval (minute)",
		Validate: minutesValidate,
	}

	interval, err := between.Run()
	if err != nil {
		fmt.Println(red(err.Error()))
		pause()
		return
	}

	reinvestInterval, _ = strconv.Atoi(interval)
	if reinvestInterval == 0 {
		reinvestInterval = 10
	}
	walletAddress, err := chain.CheckPrivateKey(privateKey)
	if err != nil {
		fmt.Println(err.Error())
		pause()
		return
	}
	if walletAddress == "" {
		fmt.Println("Invaild Private Key")
		pause()
		return
	}

	wallet = walletAddress
	chain.PrivateKey = privateKey

	//addLiquidity(big.NewInt(275651401900), big.NewInt(8156087518776), "0x25d2e80cb6b86881fd7e07dd263fb79f4abe033c", "0xa71edc38d189767582c38a3145b5873052c3e47a", client, chain)
	farmInfo, err := chain.GetPoolInfo(farmAddress, poolID)
	if err != nil {
		log.Printf("Get Farm  err  %w ", red(err))
		pause()
		return
	}

	LpToken, err := swap.NewLpToken(farmInfo.LpToken, client)
	if err != nil {
		log.Printf("Get LpToken Info Err %w  \n", err)
	}
	address0, address1, err := LpToken.Pair()
	if err != nil {
		log.Printf("Get Token Pair Info Err %w  \n", err)
		pause()
		return
	}
	swapRouter, err := swap.NewSwapRouter(router, client, chain)
	if err != nil {
		log.Printf("Get Router  Err %w  \n", err)
		pause()
		return

	}
	factory, err := swap.NewSwapFactory(swapRouter.Factory, client, chain)
	if err != nil {
		log.Printf("Get Factory  Err %w  \n", err)
		pause()
		return
	}
	sortRes, err := factory.SwapContract.SortTokens(&bind.CallOpts{}, common.HexToAddress(address0), common.HexToAddress(address1))
	if err != nil {
		log.Printf("Sort Tokens  Err %w  \n", err)
		pause()
		return
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
	fmt.Println("Start...")
	Start(farmInfo, farmAddress, poolID, chain, rewardTokenInfo, rewardToken, tokenA, tokenB, router, client, tokenAInfo, tokenBInfo)
	timer := time.NewTimer(time.Minute * time.Duration(reinvestInterval))
	for {
		select {
		case <-timer.C:
			Start(farmInfo, farmAddress, poolID, chain, rewardTokenInfo, rewardToken, tokenA, tokenB, router, client, tokenAInfo, tokenBInfo)
			timer.Reset(time.Minute * time.Duration(reinvestInterval))
		}
	}
	

}

func Start(farmInfo *chain.PoolInfo, farmAddress string, poolID int, chain *chain.Chain, rewardTokenInfo *chain.Token, rewardToken string, tokenA, tokenB, router string, client *ethclient.Client, tokenAInfo *chain.MyTokenInfo, tokenBInfo *chain.MyTokenInfo) {
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
			return
		}
		fmt.Println(blue(fmt.Sprintf("Harvest Reward Tx: %s ", tx.Hash().String())))
		txStatus, _tx := chain.WaitForBlockCompletation(tx.Hash().String())
		if txStatus != 1 {
			fmt.Println(red("Harvest Err Tx :" + tx.Hash().String()))
			return
		}
		if txStatus != 1 {

			return
		}
		sendRewardAmountToWallet, err := chain.GetTxAmount(rewardToken, farmAddress, wallet, _tx)
		if err != nil {
			log.Println(red("Search Real Reward Amount To Wallet Error: " + err.Error()))
			return
		}
		realPendingRewardAmount = sendRewardAmountToWallet
		fmt.Println(green(utils.ToDecimal(realPendingRewardAmount, int(rewardTokenInfo.Decimals)).String() + " " + rewardTokenInfo.Symbol + " -> " + wallet))
		avg := new(big.Int).Div(realPendingRewardAmount, big.NewInt(2))
		realTokenASwapAmount := big.NewInt(0)
		if strings.ToLower(tokenA) != strings.ToLower(rewardToken) {
			sendAmountToWallet, swapTxHash, err := SwapWithRetry(avg, rewardToken, tokenA, router, 10, client, chain)
			if err != nil {
				log.Println(red("swap error " + err.Error() + " " + swapTxHash))
				return
			}
			fmt.Println(
				green(
					fmt.Sprintf(
						"Swap %s %s -> %s %s Tx: %s",
						utils.ToDecimal(avg, int(rewardTokenInfo.Decimals)).String(),
						rewardTokenInfo.Symbol,
						utils.ToDecimal(sendAmountToWallet, int(tokenAInfo.Decimals)),
						tokenAInfo.Symbol,
						swapTxHash,
					),
				),
			)
			realTokenASwapAmount = sendAmountToWallet
		} else {
			realTokenASwapAmount = avg
		}
		time.Sleep(time.Second * 2)
		realTokenBSwapAmount := big.NewInt(0)
		if strings.ToLower(tokenB) != strings.ToLower(rewardToken) {
			sendAmountToWallet, swapTxHash, err := SwapWithRetry(avg, rewardToken, tokenB, router, 10, client, chain)
			if err != nil {
				log.Println(red("swap error " + err.Error() + " " + swapTxHash))
				return
			}
			fmt.Println(
				green(
					fmt.Sprintf(
						"Swap %s %s -> %s %s Tx: %s",
						utils.ToDecimal(avg, int(rewardTokenInfo.Decimals)).String(),
						rewardTokenInfo.Symbol,
						utils.ToDecimal(sendAmountToWallet, int(tokenBInfo.Decimals)),
						tokenBInfo.Symbol,
						swapTxHash,
					),
				),

			)
			realTokenBSwapAmount = sendAmountToWallet

		} else {
			realTokenBSwapAmount = avg
		}

		addLiquidityTxHash, err := AddLiquidityRetry(realTokenASwapAmount, realTokenBSwapAmount, tokenA, tokenB, 10, router, client, chain)
		if err != nil {
			log.Println(red("Rreinvest Error: ", err.Error()+addLiquidityTxHash))
			return
		}

		lpTokenApprove, err := chain.Approve(farmInfo.LpToken, farmAddress, realTokenBSwapAmount)
		if err != nil {
			fmt.Println("Approve LP Token Error: " + red(err))

			return
		}
		if !lpTokenApprove {
			fmt.Println(red("Approve LP Token  Fail"))

			return
		}
		LpTokenBalance, err := chain.GetMyTokenInfo(farmInfo.LpToken)
		if err != nil {
			log.Printf("Get My LP Token Info  Err  %s \n", red(err))

			return
		}
		fmt.Println(green(fmt.Sprintf("%s %s -> Farm ", utils.ToDecimal(LpTokenBalance.Balance, int(LpTokenBalance.Decimals)).String(), LpTokenBalance.Symbol)))
		depTx, err := chain.Deposit(farmAddress, LpTokenBalance.Balance, poolID)
		if err != nil {
			fmt.Println(red("Deposit Error: " + err.Error()))

			return
		}
		depTxStatus, _ := chain.WaitForBlockCompletation(depTx.Hash().String())
		if depTxStatus != 1 {
			log.Println(red("Deposit Err Txh :" + depTx.Hash().String()))

			return

		} else {
			log.Println(green("Deposit Success Txh :" + depTx.Hash().String()))
		}
	}

}
func SwapWithRetry(amount *big.Int, tokenA, tokenB, router string, retryCount int, client *ethclient.Client, chain *chain.Chain) (*big.Int, string, error) {
	count := 1
	keepSwap := true
	var swapTxHash string
	for {

		if count >= retryCount {
			return nil, swapTxHash, errors.New("Swap  Too Many errors")
		}
		if count > 1 {
			fmt.Printf("Try Swap %d \n", count)
		}
		if keepSwap {
			tx, err := Swap(amount, tokenA, tokenB, router, client, chain)
			if err != nil {
				//log.Println(red("swap error " + err.Error()))
				count++
				continue
			}
			swapTxHash = tx.Hash().String()
		}
		//fmt.Println(blue(fmt.Sprintf("Swap MDX -> %s Tx: %s ", tokenBInfo.Symbol, green(swapTx.Hash().String()))))
		swapTxStatus, _tx := chain.WaitForBlockCompletation(swapTxHash)
		if swapTxStatus == 1 {
			keepSwap = false
			sendAmountToWallet, err := chain.GetTxAmount(tokenB, "", wallet, _tx)
			if err != nil {
				log.Println("Swap Error: " + err.Error())
				time.Sleep(time.Minute * 5)
				count++
				continue
			}
			return sendAmountToWallet, swapTxHash, nil
		}

		count++

	}

}
func AddLiquidityRetry(wishA *big.Int, wishB *big.Int, tokenA, tokenB string, retryCount int, router string, client *ethclient.Client, chain *chain.Chain) (string, error) {
	count := 1
	var swapTxHash string
	for {
		if count >= retryCount {
			return swapTxHash, errors.New("Swap  Too Many errors")
		}
		if count > 1 {
			fmt.Printf("Try AddLiquidity %d \n", count)
		}
		addLiquidityTx, err := addLiquidity(wishA, wishB, router, tokenA, tokenB, client, chain)
		if err != nil {
			count++
			continue
		}
		swapTxHash = addLiquidityTx.Hash().String()
		//fmt.Println(blue(fmt.Sprintf("addLiquidity Tx : %s \n", addLiquidityTx.Hash().String())))
		addLiquidityTxStatus, _ := chain.WaitForBlockCompletation(addLiquidityTx.Hash().String())
		if addLiquidityTxStatus == 1 {
			return swapTxHash, nil
		}
		count++
	}

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

func pause() {
	fmt.Print("Press Any Key to Exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	return
}
