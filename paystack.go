package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.alis.build/alog"
)

type Client struct {
	secret string
}

// Create a new paystack client. Panics if PAYSTACK_SECRET env not set.
func NewClient() *Client {
	secret := os.Getenv("PAYSTACK_SECRET")
	ctx := context.Background()
	if secret == "" {
		alog.Fatal(ctx, "PAYSTACK_SECRET env var not set")
	}
	return &Client{
		secret: secret,
	}
}

func (c *Client) request(ctx context.Context, url string, method string, req_body any, resp_body any) error {
	body := []byte{}
	var err error
	if req_body != nil {
		body, err = json.Marshal(req_body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", string(resBody))
	}
	if resp_body != nil {
		if err := json.Unmarshal(resBody, resp_body); err != nil {
			return err
		}
	}
	return nil
}

type Customer struct {
	Id           int    `json:"id"`
	Email        string `json:"email"`
	CustomerCode string `json:"customer_code"`
}

// Test if the provided credentials are valid by making a GET request to /customers.
func (c *Client) ValidateCredentials(ctx context.Context) error {
	url := "https://api.paystack.co/customer"
	return c.request(ctx, url, "GET", nil, nil)
}

// Creates a new customer with the specified email and returns the new customer's id and code.
func (c *Client) CreateCustomer(ctx context.Context, email string) (*Customer, error) {
	type CreateCustomerResp struct {
		Data *Customer `json:"data"`
	}
	url := "https://api.paystack.co/customer"
	reqBody := &Customer{Email: email}
	respBody := &CreateCustomerResp{}
	err := c.request(ctx, url, "POST", reqBody, respBody)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

type InitializedTransaction struct {
	Reference        string `json:"reference"`
	AuthorizationUrl string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
}

// Initializes a new transaction for the customer with the given email.
// Amount is in the smallest unit, e.g. cents instead of ZAR.
func (c *Client) InitializeTransaction(ctx context.Context, email string, amount int32) (*InitializedTransaction, error) {
	type InitTransactionReq struct {
		Email  string `json:"email"`
		Amount string `json:"amount"`
	}
	type InitTransactionResp struct {
		Data *InitializedTransaction
	}
	url := "https://api.paystack.co/transaction/initialize"
	reqBody := &InitTransactionReq{Email: email, Amount: fmt.Sprintf("%d", amount)}
	respBody := &InitTransactionResp{}
	err := c.request(ctx, url, "POST", reqBody, respBody)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

// Charges the customer with the given email with one of their existing authorization codes.
func (c *Client) ChargeAuthorization(ctx context.Context, email string, amount int32, authCode string) (*InitializedTransaction, error) {
	type ChargeTransactionReq struct {
		Email             string `json:"email"`
		Amount            string `json:"amount"`
		AuthorizationCode string `json:"authorization_code"`
	}
	type ChargeTransactionResp struct {
		Data *InitializedTransaction
	}
	url := "https://api.paystack.co/transaction/charge_authorization"
	reqBody := &ChargeTransactionReq{Email: email, Amount: fmt.Sprintf("%d", amount), AuthorizationCode: authCode}
	respBody := &ChargeTransactionResp{}
	err := c.request(ctx, url, "POST", reqBody, respBody)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

type Authorization struct {
	AuthorizationCode string `json:"authorization_code"`
	Bin               string `json:"bin"`
	Last4             string `json:"last4"`
	ExpMonth          string `json:"exp_month"`
	ExpYear           string `json:"exp_year"`
	Channel           string `json:"channel"`
	CardType          string `json:"card_type"`
	Bank              string `json:"bank"`
	CountryCode       string `json:"country_code"`
	Brand             string `json:"brand"`
	Reusable          bool   `json:"reusable"`
	Signature         string `json:"signature"`
	AccountName       string `json:"account_name"`
}

type VerifiedTransaction struct {
	Id            int            `json:"id"`
	Reference     string         `json:"reference"`
	Status        string         `json:"status"`
	Authorization *Authorization `json:"authorization"`
}

// Verifies a transaction with the given reference. The returned status could be "success", "failed", or anything else indicating its pending.
func (c *Client) VerifyTransaction(ctx context.Context, ref string) (*VerifiedTransaction, error) {
	type VerifiedTransactionResp struct {
		Data *VerifiedTransaction
	}
	url := "https://api.paystack.co/transaction/verify/" + ref
	resp := &VerifiedTransactionResp{}
	if err := c.request(ctx, url, "GET", nil, resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
