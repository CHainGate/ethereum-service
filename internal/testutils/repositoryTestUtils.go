package testutils

import (
	"ethereum-service/model"
	"log"
	"time"

	"github.com/CHainGate/backend/pkg/enum"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	ChaingateAcc     *model.Account
	MerchantAcc      *model.Account
	EmptyPayment     model.Payment
	WaitingPayment   model.Payment
	PartiallyPayment model.Payment
)

func getEmptyPayment(acc *model.Account, mAcc *model.Account) model.Payment {
	return model.Payment{
		Account:       acc,
		Mode:          "Test",
		Base:          model.Base{ID: uuid.New()},
		AccountID:     acc.ID,
		PriceAmount:   100,
		PriceCurrency: "USD",
		UserWallet:    mAcc.Address,
	}
}

func addWaitingPaymentState(payment model.Payment) model.Payment {
	state := model.PaymentState{
		Base:           model.Base{ID: uuid.New()},
		StatusName:     "waiting",
		AccountID:      payment.AccountID,
		AmountReceived: model.NewBigIntFromInt(0),
		PayAmount:      model.NewBigIntFromInt(1000000),
		PaymentID:      payment.ID,
	}
	payment.CurrentPaymentStateId = &state.ID
	payment.CurrentPaymentState = state
	payment.PaymentStates = append(payment.PaymentStates, state)
	return payment
}

func addPartiallyPaidPaymentState(payment model.Payment) model.Payment {
	state := model.PaymentState{
		Base:           model.Base{ID: uuid.New()},
		StatusName:     enum.StatePartiallyPaid.String(),
		AccountID:      payment.AccountID,
		AmountReceived: model.NewBigIntFromInt(10),
		PayAmount:      model.NewBigIntFromInt(1000000),
		PaymentID:      payment.ID,
	}
	payment.CurrentPaymentStateId = &state.ID
	payment.CurrentPaymentState = state
	payment.PaymentStates = append(payment.PaymentStates, state)
	return payment
}

func Setup() {
	ChaingateAcc = model.CreateAccount()
	MerchantAcc = model.CreateAccount()
	ChaingateAcc.ID = uuid.New()
	EmptyPayment = getEmptyPayment(ChaingateAcc, MerchantAcc)
	WaitingPayment = addWaitingPaymentState(EmptyPayment)
	PartiallyPayment = addPartiallyPaidPaymentState(WaitingPayment)
}

func getPaymentRow(p model.Payment) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "user_wallet", "mode", "price_amount", "price_currency", "current_payment_state_id"}).
		AddRow(p.ID, MerchantAcc.Address, "Test", "100", "USD", p.CurrentPaymentStateId)
}

func getAccountRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "private_key", "address", "nonce", "used", "remainder"}).
		AddRow(ChaingateAcc.ID, time.Now(), time.Now(), time.Now(), ChaingateAcc.PrivateKey, ChaingateAcc.Address, ChaingateAcc.Nonce, true, ChaingateAcc.Remainder)
}

func getFreeAccountRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "private_key", "address", "nonce", "used", "remainder"}).
		AddRow(ChaingateAcc.ID, time.Now(), time.Now(), time.Now(), ChaingateAcc.PrivateKey, ChaingateAcc.Address, ChaingateAcc.Nonce, false, ChaingateAcc.Remainder)
}

func getPaymentStatesRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "account_id", "pay_amount", "amount_received", "status_name", "payment_id"}).
		AddRow(WaitingPayment.CurrentPaymentStateId, time.Now(), time.Now(), time.Now(), ChaingateAcc.ID, WaitingPayment.CurrentPaymentState.PayAmount, WaitingPayment.CurrentPaymentState.AmountReceived, WaitingPayment.CurrentPaymentState.StatusName, WaitingPayment.ID)
}

func SetupCreatePayment(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	paymentRows := getPaymentRow(EmptyPayment)
	accRows := getAccountRow()
	stateRows := getPaymentStatesRow()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ChaingateAcc.PrivateKey, ChaingateAcc.Address, ChaingateAcc.Nonce, true, ChaingateAcc.Remainder, ChaingateAcc.ID).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ChaingateAcc.ID, WaitingPayment.CurrentPaymentState.PayAmount, "0", enum.StateWaiting.String(), WaitingPayment.ID).
		WillReturnRows(stateRows)
	mock.ExpectQuery("INSERT INTO \"payments\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ChaingateAcc.ID, MerchantAcc.Address, EmptyPayment.Mode, EmptyPayment.PriceAmount, EmptyPayment.PriceCurrency, WaitingPayment.CurrentPaymentStateId, WaitingPayment.ID).
		WillReturnRows(paymentRows)
	mock.ExpectCommit()
	return mock
}

func SetupAllPaymentIntents(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	paymentRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "account_id", "user_wallet", "mode", "price_amount", "price_currency", "current_payment_state_id",
		"CurrentPaymentState__id", "CurrentPaymentState__created_at", "CurrentPaymentState__updated_at", "CurrentPaymentState__deleted_at", "CurrentPaymentState__account_id", "CurrentPaymentState__pay_amount",
		"CurrentPaymentState__amount_received", "CurrentPaymentState__status_name", "CurrentPaymentState__payment_id"}).
		AddRow(WaitingPayment.ID, time.Now(), time.Now(), time.Now(), ChaingateAcc.ID, MerchantAcc.Address, WaitingPayment.Mode, WaitingPayment.PriceAmount, WaitingPayment.PriceCurrency, WaitingPayment.CurrentPaymentStateId,
			WaitingPayment.CurrentPaymentStateId, time.Now(), time.Now(), time.Now(), ChaingateAcc.ID, WaitingPayment.CurrentPaymentState.PayAmount, WaitingPayment.CurrentPaymentState.AmountReceived, WaitingPayment.CurrentPaymentState.StatusName, WaitingPayment.ID)

	mock.ExpectQuery("SELECT (.+) FROM \"payments\"").
		WithArgs("waiting", "partially_paid").
		WillReturnRows(paymentRows)

	accRows := getAccountRow()
	mock.ExpectQuery("SELECT (.+) FROM \"accounts\"").
		WithArgs(ChaingateAcc.ID).
		WillReturnRows(accRows)

	stateRows := getPaymentStatesRow()
	mock.ExpectQuery("SELECT (.+) FROM \"payment_states\"").
		WithArgs(WaitingPayment.CurrentPaymentStateId).
		WillReturnRows(stateRows)

	return mock
}

func SetupUpdatePaymentState(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	stateRows := getPaymentStatesRow()
	accRows := getAccountRow()

	mock.ExpectBegin()

	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ChaingateAcc.PrivateKey, ChaingateAcc.Address, ChaingateAcc.Nonce, true, ChaingateAcc.Remainder, ChaingateAcc.ID).
		WillReturnRows(accRows)
	mock.ExpectQuery("INSERT INTO \"payment_states\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ChaingateAcc.ID, PartiallyPayment.CurrentPaymentState.PayAmount, PartiallyPayment.CurrentPaymentState.AmountReceived, PartiallyPayment.CurrentPaymentState.StatusName, PartiallyPayment.ID).
		WillReturnRows(stateRows)
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ChaingateAcc.ID, PartiallyPayment.UserWallet, PartiallyPayment.Mode, PartiallyPayment.PriceAmount, PartiallyPayment.PriceCurrency, WaitingPayment.CurrentPaymentStateId, PartiallyPayment.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	return mock
}

func SetupCreateAccount(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	accRows := getAccountRow()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ChaingateAcc.PrivateKey, ChaingateAcc.Address, ChaingateAcc.Nonce, true, ChaingateAcc.Remainder, ChaingateAcc.ID).
		WillReturnRows(accRows)
	mock.ExpectCommit()
	return mock
}

func SetupUpdateAccount(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE \"accounts\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), ChaingateAcc.PrivateKey, ChaingateAcc.Address, ChaingateAcc.Nonce, true, ChaingateAcc.Remainder, ChaingateAcc.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	return mock
}

func SetupGetFreeAccount(mock sqlmock.Sqlmock) sqlmock.Sqlmock {
	accRows := getFreeAccountRow()

	mock.ExpectQuery("SELECT (.+) FROM \"accounts\"").
		WithArgs("false").
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
