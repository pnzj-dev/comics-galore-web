package nowpayments

type CurrenciesResponse struct {
	Currencies []string `json:"currencies"`
}
type StatusResponse struct {
	Message string `json:"message"`
}

type Request struct {
	PriceAmount      float32 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	PayAmount        float32 `json:"pay_amount,omitempty"`
	PayCurrency      string  `json:"pay_currency"`
	IPNCallbackURL   string  `json:"ipn_callback_url,omitempty"`
	OrderID          string  `json:"order_id,omitempty"`
	OrderDescription string  `json:"order_description,omitempty"`
	Case             string  `json:"case,omitempty"`
}

type Response struct {
	PaymentID        string  `json:"payment_id"`
	PaymentStatus    string  `json:"payment_status"`
	PayAddress       string  `json:"pay_address"`
	PriceAmount      float32 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	PayAmount        float32 `json:"pay_amount"`
	PayCurrency      string  `json:"pay_currency"`
	OrderID          string  `json:"order_id"`
	OrderDescription string  `json:"order_description"`
	IPNCallbackURL   string  `json:"ipn_callback_url"`
}

type EstimatedPrice struct {
	CurrencyFrom    string  `json:"currency_from"`
	CurrencyTo      string  `json:"currency_to"`
	EstimatedAmount float32 `json:"estimated_amount"`
}

type Status string

const (
	StatusWaiting       Status = "waiting"
	StatusConfirming    Status = "confirming"
	StatusConfirmed     Status = "confirmed"
	StatusSending       Status = "sending"
	StatusPartiallyPaid Status = "partially_paid"
	StatusFinished      Status = "finished"
	StatusFailed        Status = "failed"
	StatusRefunded      Status = "refunded"
	StatusExpired       Status = "expired"
)

type PaymentStatus struct {
	PaymentID       string  `json:"payment_id"`
	PaymentStatus   Status  `json:"payment_status"`
	PayAddress      string  `json:"pay_address"`
	PriceAmount     float32 `json:"price_amount"`
	PriceCurrency   string  `json:"price_currency"`
	PayAmount       float32 `json:"pay_amount"`
	ActuallyPaid    float32 `json:"actually_paid"`
	PayCurrency     string  `json:"pay_currency"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	PurchaseID      string  `json:"purchase_id"`
	OutcomeCurrency string  `json:"outcome_currency"`
	OutcomeAmount   float32 `json:"outcome_amount"`
}
