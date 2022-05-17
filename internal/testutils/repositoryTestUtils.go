package testutils

import (
	"ethereum-service/model"
	"ethereum-service/utils"
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
	partiallyPayment *model.Payment
	paidPayment      *model.Payment
	forwardedPayment *model.Payment
	finishedPayment  *model.Payment
)

func createEmptyPayment(acc model.Account, mAcc model.Account) *model.Payment {
	return &model.Payment{
		Account:       &acc,
		Mode:          "main",
		Base:          model.Base{ID: uuid.New()},
		AccountID:     acc.ID,
		PriceAmount:   100,
		PriceCurrency: "USD",
		UserWallet:    mAcc.Address,
	}
}

func CreatePaymentState(paymentID uuid.UUID, accountID uuid.UUID, state enum.State, amountReceived *big.Int) model.PaymentState {
	return model.PaymentState{
		Base:           model.Base{ID: uuid.New()},
		StatusName:     state.String(),
		AccountID:      paymentID,
		AmountReceived: model.NewBigInt(amountReceived),
		PayAmount:      model.NewBigIntFromInt(100000000000000),
		PaymentID:      accountID,
	}
}

func addWaitingPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Waiting, big.NewInt(0))
	payment.CurrentPaymentStateId = &state.ID
	payment.CurrentPaymentState = state
	payment.PaymentStates = append(payment.PaymentStates, state)
	return &payment
}

func addPartiallyPaidPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.PartiallyPaid, big.NewInt(10))
	payment.CurrentPaymentStateId = &state.ID
	payment.CurrentPaymentState = state
	payment.PaymentStates = append(payment.PaymentStates, state)
	return &payment
}

func addPaidPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Paid, big.NewInt(100000000000000))
	payment.CurrentPaymentStateId = &state.ID
	payment.CurrentPaymentState = state
	payment.PaymentStates = append(payment.PaymentStates, state)
	return &payment
}

func addForwardedPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Forwarded, big.NewInt(100000000000000))
	payment.CurrentPaymentStateId = &state.ID
	payment.CurrentPaymentState = state
	payment.PaymentStates = append(payment.PaymentStates, state)
	return &payment
}

func addFinishedPaymentState(payment model.Payment) *model.Payment {
	state := CreatePaymentState(payment.ID, payment.AccountID, enum.Finished, big.NewInt(100000000000000))
	payment.CurrentPaymentStateId = &state.ID
	payment.CurrentPaymentState = state
	payment.PaymentStates = append(payment.PaymentStates, state)
	return &payment
}

func GetChaingateAcc() model.Account {
	if chaingateAcc == nil {
		chaingateAcc = model.CreateAccount(enum.Main)
		chaingateAcc.ID = uuid.New()
	}
	return *chaingateAcc
}

func GetMerchantAcc() model.Account {
	if merchantAcc == nil {
		merchantAcc = model.CreateAccount(enum.Main)
		merchantAcc.ID = uuid.New()
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

func GetPartiallyPayment() model.Payment {
	partiallyPayment = addPartiallyPaidPaymentState(GetWaitingPayment())
	return *partiallyPayment
}

func GetPaidPayment() model.Payment {
	paidPayment = addPaidPaymentState(GetWaitingPayment())
	return *paidPayment
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
	return sqlmock.NewRows([]string{"id", "user_wallet", "mode", "price_amount", "price_currency", "current_payment_state_id"}).
		AddRow(p.ID, GetMerchantAcc().Address, "main", "100", "USD", p.CurrentPaymentStateId)
}

func getAccountRow(a model.Account) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "private_key", "address", "nonce", "used", "remainder", "mode"}).
		AddRow(a.ID, time.Now(), time.Now(), time.Now(), a.PrivateKey, a.Address, a.Nonce, true, a.Remainder, a.Mode)
}

func getFreeAccountRow(a model.Account) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "private_key", "address", "nonce", "used", "remainder", "mode"}).
		AddRow(a.ID, time.Now(), time.Now(), time.Now(), a.PrivateKey, a.Address, a.Nonce, false, a.Remainder, a.Mode)
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
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, wp.CurrentPaymentState.PayAmount, "0", enum.Waiting.String(), sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectQuery("INSERT INTO \"payments\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, ma.Address, ep.Mode, ep.PriceAmount, ep.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg()).
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
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, wp.CurrentPaymentState.PayAmount, "0", enum.Waiting.String(), sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectQuery("INSERT INTO \"payments\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, sqlmock.AnyArg(), ep.Mode, ep.PriceAmount, ep.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(paymentRows)
	mock.ExpectCommit()
	return mock
}

func SetupAllPaymentIntents(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	wp := GetWaitingPayment()
	ma := GetMerchantAcc()
	ca := GetChaingateAcc()
	paymentRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "account_id", "user_wallet", "mode", "price_amount", "price_currency", "current_payment_state_id",
		"CurrentPaymentState__id", "CurrentPaymentState__created_at", "CurrentPaymentState__updated_at", "CurrentPaymentState__deleted_at", "CurrentPaymentState__account_id", "CurrentPaymentState__pay_amount",
		"CurrentPaymentState__amount_received", "CurrentPaymentState__status_name", "CurrentPaymentState__payment_id"}).
		AddRow(wp.ID, time.Now(), time.Now(), time.Now(), ca.ID, ma.Address, wp.Mode, wp.PriceAmount, wp.PriceCurrency, wp.CurrentPaymentStateId,
			wp.CurrentPaymentStateId, time.Now(), time.Now(), time.Now(), ca.ID, wp.CurrentPaymentState.PayAmount, wp.CurrentPaymentState.AmountReceived, wp.CurrentPaymentState.StatusName, wp.ID)

	mock.ExpectQuery("SELECT (.+) FROM \"payments\"").
		WithArgs("waiting", "partially_paid").
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
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, true, ca.Remainder, ca.Mode, sqlmock.AnyArg()).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.CurrentPaymentState.PayAmount, pp.CurrentPaymentState.AmountReceived, pp.CurrentPaymentState.StatusName, sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.UserWallet, pp.Mode, pp.PriceAmount, pp.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg()).
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

	mock.ExpectBegin()

	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, true, ca.Remainder, ca.Mode, sqlmock.AnyArg()).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.CurrentPaymentState.PayAmount, model.NewBigInt(amountPaid), pp.CurrentPaymentState.StatusName, sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.UserWallet, pp.Mode, pp.PriceAmount, pp.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	return mock
}

func SetupUpdatePaymentStateToForwarded(mock sqlmock.Sqlmock, amountPaid *big.Int) sqlmock.Sqlmock {
	fp := GetForwardedPayment()
	ca := GetChaingateAcc()
	ca.Nonce = ca.Nonce + 1
	amountAfterPayment := big.NewInt(0).Sub(amountPaid, &fp.CurrentPaymentState.PayAmount.Int)
	ca.Remainder = model.NewBigInt(big.NewInt(0).Add(amountAfterPayment, utils.GetChaingateEarnings(&fp.CurrentPaymentState.PayAmount.Int)))
	stateRows := getPaymentStatesRow(ca, fp)
	accRows := getAccountRow(ca)
	mock.ExpectBegin()

	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, true, ca.Remainder, ca.Mode, sqlmock.AnyArg()).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, fp.CurrentPaymentState.PayAmount, model.NewBigInt(amountPaid), fp.CurrentPaymentState.StatusName, sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, fp.UserWallet, fp.Mode, fp.PriceAmount, fp.PriceCurrency, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	return mock
}

func SetupUpdatePaymentStateToFinished(mock sqlmock.Sqlmock, amountPaid *big.Int, nonce uint64, remainder *model.BigInt) sqlmock.Sqlmock {
	pp := GetFinishedPayment()
	pp.CurrentPaymentState.AmountReceived = model.NewBigInt(amountPaid)
	ca := GetChaingateAcc()
	ca.Remainder = remainder
	stateRows := getPaymentStatesRow(ca, pp)
	ca.Nonce = nonce
	accRows := getAccountRow(ca)

	mock.ExpectBegin()

	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.PrivateKey, ca.Address, ca.Nonce, false, ca.Remainder, ca.Mode, sqlmock.AnyArg()).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.CurrentPaymentState.PayAmount, model.NewBigInt(amountPaid), pp.CurrentPaymentState.StatusName, sqlmock.AnyArg()).
		WillReturnRows(stateRows)
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ca.ID, pp.UserWallet, pp.Mode, pp.PriceAmount, pp.PriceCurrency, pp.CurrentPaymentStateId, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

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
	accRows := getFreeAccountRow(ca)

	mock.ExpectQuery("SELECT (.+) FROM \"accounts\"").
		WithArgs("false", "main").
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
