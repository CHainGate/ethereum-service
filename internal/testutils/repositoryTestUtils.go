package testutils

import (
	"context"
	"ethereum-service/internal/config"
	"ethereum-service/model"
	"ethereum-service/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"time"

	"github.com/CHainGate/backend/pkg/enum"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	chaingateAcc     *model.Account
	merchantAcc      *model.Account
	emptyPayment     *model.Payment
	waitingPayment   *model.Payment
	expiredPayment   *model.Payment
	partiallyPayment *model.Payment
	paidPayment      *model.Payment
	confirmedPayment *model.Payment
	forwardedPayment *model.Payment
	finishedPayment  *model.Payment
)

func createEmptyPayment(acc model.Account, mAcc model.Account) *model.Payment {
	return &model.Payment{
		Account: &acc,
		Mode:    enum.Main,
		Base: model.Base{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		AccountID:            acc.ID,
		PriceAmount:          100,
		PriceCurrency:        "USD",
		ForwardingBlockNr:    model.NewBigIntFromInt(0),
		LastReceivingBlockNr: model.NewBigIntFromInt(0),
		MerchantWallet:       mAcc.Address,
	}
}

func CreatePaymentState(paymentID uuid.UUID, accountID uuid.UUID, state enum.State, amountReceived *big.Int) model.PaymentState {
	return model.PaymentState{
		Base:           model.Base{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		StatusName:     state,
		AccountID:      paymentID,
		AmountReceived: model.NewBigInt(amountReceived),
		PayAmount:      model.NewBigIntFromInt(100000000000000),
		PaymentID:      accountID,
	}
}

func addPaymentState(payment model.Payment, state model.PaymentState) *model.Payment {
	payment.CurrentPaymentStateId = &state.ID
	payment.CurrentPaymentState = state
	payment.PaymentStates = append(payment.PaymentStates, state)
	return &payment
}

func addWaitingPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Waiting, big.NewInt(0))
	return addPaymentState(payment, state)
}

func addExpiredPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Expired, big.NewInt(0))
	return addPaymentState(payment, state)
}

func addPartiallyPaidPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.PartiallyPaid, big.NewInt(10))
	return addPaymentState(payment, state)
}

func addPaidPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Paid, big.NewInt(100000000000000))
	return addPaymentState(payment, state)
}

func addConfirmedPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Confirmed, big.NewInt(100000000000000))
	return addPaymentState(payment, state)
}

func addForwardedPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Forwarded, big.NewInt(100000000000000))
	return addPaymentState(payment, state)
}

func addFinishedPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Finished, big.NewInt(100000000000000))
	return addPaymentState(payment, state)
}

func addFailedPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Failed, big.NewInt(0))
	return addPaymentState(payment, state)
}

func GetChaingateAcc() model.Account {
	if chaingateAcc == nil {
		chaingateAcc = model.CreateAccount(enum.Main)
		chaingateAcc.ID = uuid.New()
		chaingateAcc.CreatedAt = time.Now()
		chaingateAcc.UpdatedAt = time.Now()
	}
	return *chaingateAcc
}

func GetMerchantAcc() model.Account {
	if merchantAcc == nil {
		merchantAcc = model.CreateAccount(enum.Main)
		merchantAcc.ID = uuid.New()
		chaingateAcc.CreatedAt = time.Now()
		chaingateAcc.UpdatedAt = time.Now()
	}
	return *merchantAcc
}

func GetEmptyPayment() model.Payment {
	emptyPayment = createEmptyPayment(GetChaingateAcc(), GetMerchantAcc())
	return *emptyPayment
}

func GetWaitingPayment() model.Payment {
	waitingPayment = addWaitingPaymentState(GetEmptyPayment())
	return *waitingPayment
}

func GetExpiredPayment() model.Payment {
	expiredPayment = addExpiredPaymentState(GetWaitingPayment())
	return *expiredPayment
}

func GetPartiallyPayment() model.Payment {
	partiallyPayment = addPartiallyPaidPaymentState(GetWaitingPayment())
	return *partiallyPayment
}

func GetPaidPayment() model.Payment {
	paidPayment = addPaidPaymentState(GetWaitingPayment())
	return *paidPayment
}

func GetConfirmedPayment() model.Payment {
	confirmedPayment = addConfirmedPaymentState(GetPaidPayment())
	return *confirmedPayment
}

func GetForwardedPayment() model.Payment {
	forwardedPayment = addForwardedPaymentState(GetPaidPayment())
	return *forwardedPayment
}

func GetFinishedPayment() model.Payment {
	finishedPayment = addFinishedPaymentState(GetPaidPayment())
	return *finishedPayment
}

func getPaymentRow(p model.Payment) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "merchant_wallet", "mode", "price_amount", "price_currency", "current_payment_state_id", "forwarding_transaction_hash", "receiving_block_nr", "forwarding_block_nr"}).
		AddRow(p.ID, GetMerchantAcc().Address, 1, "100", "USD", p.CurrentPaymentStateId, p.ForwardingTransactionHash, p.LastReceivingBlockNr, p.ForwardingBlockNr)
}

func getAccountRow(a model.Account) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "private_key", "address", "nonce", "used", "remainder", "mode"}).
		AddRow(a.ID, time.Now(), time.Now(), time.Now(), a.PrivateKey, a.Address, a.Nonce, a.Used, a.Remainder, a.Mode)
}

func getPaymentStatesRow(a model.Account, p model.Payment) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "account_id", "pay_amount", "amount_received", "status_name", "payment_id"}).
		AddRow(p.CurrentPaymentStateId, time.Now(), time.Now(), time.Now(), a.ID, p.CurrentPaymentState.PayAmount, p.CurrentPaymentState.AmountReceived, p.CurrentPaymentState.StatusName, p.ID)
}

func SetupCreatePayment(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	ep := GetEmptyPayment()
	wp := GetWaitingPayment()
	ma := GetMerchantAcc()
	ca := GetChaingateAcc()
	paymentRows := getPaymentRow(ep)
	accRows := getAccountRow(ca)
	stateRows := getPaymentStatesRow(ca, wp)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, true, ca.Remainder, ca.Mode, ca.ID).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, wp.CurrentPaymentState.PayAmount, "0", enum.Waiting, sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectQuery("INSERT INTO \"payments\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, ma.Address, ep.Mode, ep.PriceAmount, ep.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(paymentRows)
	mock.ExpectCommit()
	return mock
}

func SetupCreatePaymentWithoutIdCheck(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	ep := GetEmptyPayment()
	wp := GetWaitingPayment()
	ca := GetChaingateAcc()
	paymentRows := getPaymentRow(ep)
	accRows := getAccountRow(ca)
	stateRows := getPaymentStatesRow(ca, wp)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, true, ca.Remainder, ca.Mode, ca.ID).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, wp.CurrentPaymentState.PayAmount, "0", enum.Waiting, sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectQuery("INSERT INTO \"payments\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, sqlmock.AnyArg(), ep.Mode, ep.PriceAmount, ep.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(paymentRows)
	mock.ExpectCommit()
	return mock
}

func SetupAllPayments(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	wp := GetWaitingPayment()
	ma := GetMerchantAcc()
	ca := GetChaingateAcc()
	paymentRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "account_id", "merchant_wallet", "mode", "price_amount", "price_currency", "current_payment_state_id",
		"CurrentPaymentState__id", "CurrentPaymentState__created_at", "CurrentPaymentState__updated_at", "CurrentPaymentState__deleted_at", "CurrentPaymentState__account_id", "CurrentPaymentState__pay_amount",
		"CurrentPaymentState__amount_received", "CurrentPaymentState__status_name", "CurrentPaymentState__payment_id"}).
		AddRow(wp.ID, time.Now(), time.Now(), time.Now(), ca.ID, ma.Address, wp.Mode, wp.PriceAmount, wp.PriceCurrency, wp.CurrentPaymentStateId,
			wp.CurrentPaymentStateId, time.Now(), time.Now(), time.Now(), ca.ID, wp.CurrentPaymentState.PayAmount, wp.CurrentPaymentState.AmountReceived, wp.CurrentPaymentState.StatusName, wp.ID)

	mock.ExpectQuery("SELECT (.+) FROM \"payments\"").
		WithArgs(enum.Waiting, enum.PartiallyPaid).
		WillReturnRows(paymentRows)

	accRows := getAccountRow(ca)
	mock.ExpectQuery("SELECT (.+) FROM \"accounts\"").
		WithArgs(chaingateAcc.ID).
		WillReturnRows(accRows)

	stateRows := getPaymentStatesRow(ca, wp)
	mock.ExpectQuery("SELECT (.+) FROM \"payment_states\"").
		WithArgs(wp.CurrentPaymentStateId).
		WillReturnRows(stateRows)

	stateRows = getPaymentStatesRow(ca, wp)
	mock.ExpectQuery("SELECT (.+) FROM \"payment_states\"").
		WithArgs(wp.ID).
		WillReturnRows(stateRows)

	return mock
}

func SetupModePayments(mock sqlmock.Sqlmock, mode enum.Mode, state enum.State) sqlmock.Sqlmock {
	wp := GetWaitingPayment()
	ma := GetMerchantAcc()
	ca := GetChaingateAcc()
	paymentRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "account_id", "merchant_wallet", "mode", "price_amount", "price_currency", "current_payment_state_id",
		"CurrentPaymentState__id", "CurrentPaymentState__created_at", "CurrentPaymentState__updated_at", "CurrentPaymentState__deleted_at", "CurrentPaymentState__account_id", "CurrentPaymentState__pay_amount",
		"CurrentPaymentState__amount_received", "CurrentPaymentState__status_name", "CurrentPaymentState__payment_id"}).
		AddRow(wp.ID, time.Now(), time.Now(), time.Now(), ca.ID, ma.Address, wp.Mode, wp.PriceAmount, wp.PriceCurrency, wp.CurrentPaymentStateId,
			wp.CurrentPaymentStateId, time.Now(), time.Now(), time.Now(), ca.ID, wp.CurrentPaymentState.PayAmount, wp.CurrentPaymentState.AmountReceived, wp.CurrentPaymentState.StatusName, wp.ID)

	mock.ExpectQuery("SELECT (.+) FROM \"payments\"").
		WithArgs(mode, state).
		WillReturnRows(paymentRows)

	accRows := getAccountRow(ca)
	mock.ExpectQuery("SELECT (.+) FROM \"accounts\"").
		WithArgs(chaingateAcc.ID).
		WillReturnRows(accRows)

	stateRows := getPaymentStatesRow(ca, wp)
	mock.ExpectQuery("SELECT (.+) FROM \"payment_states\"").
		WithArgs(wp.CurrentPaymentStateId).
		WillReturnRows(stateRows)

	return mock
}

func SetupUpdatePaymentState(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	pp := GetPartiallyPayment()
	ca := GetChaingateAcc()
	stateRows := getPaymentStatesRow(ca, pp)
	accRows := getAccountRow(ca)

	mock.ExpectBegin()

	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, ca.Used, ca.Remainder, ca.Mode, sqlmock.AnyArg()).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.CurrentPaymentState.PayAmount, pp.CurrentPaymentState.AmountReceived, pp.CurrentPaymentState.StatusName, sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.MerchantWallet, pp.Mode, pp.PriceAmount, pp.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	return mock
}

func SetupUpdatePaymentStateToPaid(mock sqlmock.Sqlmock, amountPaid *big.Int) sqlmock.Sqlmock {
	pp := GetPaidPayment()
	pp.CurrentPaymentState.AmountReceived = model.NewBigInt(amountPaid)
	ca := GetChaingateAcc()
	stateRows := getPaymentStatesRow(ca, pp)
	accRows := getAccountRow(ca)

	mockRequests(mock, amountPaid, ca, accRows, pp, stateRows)

	return mock
}

func SetupUpdatePaymentStateToConfirmed(mock sqlmock.Sqlmock, amountPaid *big.Int) sqlmock.Sqlmock {
	pp := GetConfirmedPayment()
	pp.CurrentPaymentState.AmountReceived = model.NewBigInt(amountPaid)
	ca := GetChaingateAcc()
	stateRows := getPaymentStatesRow(ca, pp)
	accRows := getAccountRow(ca)

	mockRequests(mock, amountPaid, ca, accRows, pp, stateRows)

	return mock
}

func SetupUpdatePaymentStateToExpired(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	ep := GetExpiredPayment()
	ca := GetChaingateAcc()
	ca.Used = false
	stateRows := getPaymentStatesRow(ca, ep)
	accRows := getAccountRow(ca)

	mockRequests(mock, big.NewInt(0), ca, accRows, ep, stateRows)

	return mock
}

func SetupUpdatePaymentStateToFailed(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	fp := addFailedPaymentState(GetPaidPayment())
	ca := GetChaingateAcc()
	ca.Used = false
	stateRows := getPaymentStatesRow(ca, *fp)
	accRows := getAccountRow(ca)

	mockRequests(mock, big.NewInt(0), ca, accRows, *fp, stateRows)

	return mock
}

func mockRequests(mock sqlmock.Sqlmock, amountPaid *big.Int, ca model.Account, accRows *sqlmock.Rows, pp model.Payment, stateRows *sqlmock.Rows) {
	mock.ExpectBegin()

	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, ca.Used, ca.Remainder, ca.Mode, sqlmock.AnyArg()).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.CurrentPaymentState.PayAmount, model.NewBigInt(amountPaid), pp.CurrentPaymentState.StatusName, sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.MerchantWallet, pp.Mode, pp.PriceAmount, pp.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
}

func SetupUpdatePaymentStateToForwarded(mock sqlmock.Sqlmock, amountPaid *big.Int) sqlmock.Sqlmock {
	fp := GetForwardedPayment()
	ca := GetChaingateAcc()
	ca.Nonce = ca.Nonce + 1
	fp.CurrentPaymentState.AmountReceived = model.NewBigInt(amountPaid)
	amountAfterPayment := big.NewInt(0).Sub(amountPaid, &fp.CurrentPaymentState.PayAmount.Int)
	ca.Remainder = model.NewBigInt(big.NewInt(0).Add(amountAfterPayment, utils.GetChaingateEarnings(&fp.CurrentPaymentState.PayAmount.Int)))
	stateRows := getPaymentStatesRow(ca, fp)
	accRows := getAccountRow(ca)
	mockRequests(mock, amountPaid, ca, accRows, fp, stateRows)

	return mock
}

func SetupUpdatePaymentStateToFinished(mock sqlmock.Sqlmock, amountPaid *big.Int, nonce uint64, remainder *model.BigInt) sqlmock.Sqlmock {
	pp := GetFinishedPayment()
	pp.CurrentPaymentState.AmountReceived = model.NewBigInt(amountPaid)
	ca := GetChaingateAcc()
	ca.Remainder = remainder
	stateRows := getPaymentStatesRow(ca, pp)
	ca.Nonce = nonce
	ca.Used = false
	accRows := getAccountRow(ca)

	mockRequests(mock, amountPaid, ca, accRows, pp, stateRows)
	return mock
}

func SetupCreateAccount(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	ca := GetChaingateAcc()
	accRows := getAccountRow(ca)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, true, ca.Remainder, ca.Mode, sqlmock.AnyArg()).
		WillReturnRows(accRows)
	mock.ExpectCommit()
	return mock
}

func SetupUpdateAccount(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	ca := GetChaingateAcc()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, true, ca.Remainder, ca.Mode, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	return mock
}

func SetupUpdateAccountFree(mock sqlmock.Sqlmock, nonce uint64) sqlmock.Sqlmock {
	ca := GetChaingateAcc()
	ca.Nonce = nonce
	ca.Used = false
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, ca.Used, sqlmock.AnyArg(), ca.Mode, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	return mock
}

func SetupGetFreeAccount(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	ca := GetChaingateAcc()
	ca.Used = false
	accRows := getAccountRow(ca)

	mock.ExpectQuery("SELECT (.+) FROM \"accounts\"").
		WithArgs("false", 1).
		WillReturnRows(accRows)
	return mock
}

func NewMock() (sqlmock.Sqlmock, *gorm.DB) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	dialector := postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	})

	gormDb, err := gorm.Open(dialector, &gorm.Config{})
	return mock, gormDb
}

func CreateInitialPayment(client *ethclient.Client, genesisAcc *model.Account, payAmount *big.Int, targetAddress string) *types.Transaction {
	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(genesisAcc.Address))
	if err != nil {
		log.Fatal(err)
	}

	var gasPrice *big.Int
	if config.Chain == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		gasPrice = config.Chain.GasPrice
	}
	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	var chainID *big.Int
	if config.Chain == nil {
		chainID, err = client.NetworkID(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		chainID = config.Chain.ChainId
	}

	gasLimit := uint64(21000)

	toAddress := common.HexToAddress(targetAddress)

	// Transaction fees and Gas explained: https://docs.avax.network/learn/platform-overview/transaction-fees
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: gasPrice,  //gasPrice,     // maximum price per unit of gas that the transaction is willing to pay
		GasTipCap: gasTipCap, //tipCap,       // maximum amount above the baseFee of a block that the transaction is willing to pay to be included
		Gas:       gasLimit,
		To:        &toAddress,
		Value:     payAmount,
	})

	key, err := utils.GetPrivateKey(genesisAcc.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	return signedTx
}
