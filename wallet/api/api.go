package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/log"
	"github.com/kurumiimari/gohan/wallet"
	"github.com/pkg/errors"
	"net/http"
	"os"
)

var apiLogger = log.ModuleLogger("api")

type ErrorResponse struct {
	Msg string `json:"msg"`
}

var invalidJSONRes = &ErrorResponse{
	Msg: "Mal-formed JSON payload.",
}

func UnmarshalRequestJSON(w http.ResponseWriter, r *http.Request, in interface{}) bool {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(in); err == nil {
		return true
	}
	w.WriteHeader(400)
	MarshalResponseJSON(w, invalidJSONRes)
	return false
}

func MarshalErrorJSON(w http.ResponseWriter, err error, code int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	apiLogger.Error("error handling request", "err", err)
	fmt.Fprintf(os.Stderr, "%+v", err)
	MarshalResponseJSON(w, &ErrorResponse{Msg: err.Error()})
}

func MarshalResponseJSON(w http.ResponseWriter, out interface{}) {
	data, err := json.Marshal(out)
	if err != nil {
		apiLogger.Panic("error marshaling JSON response, shutting down", "err", err)
	}
	if _, err := w.Write(data); err != nil {
		apiLogger.Warning("error writing JSON response")
	}
}

type API struct {
	network *chain.Network
	node    *wallet.Node
	apiKey  string
}

func NewAPI(network *chain.Network, service *wallet.Node, apiKey string) http.Handler {
	api := &API{
		network: network,
		node:    service,
		apiKey:  apiKey,
	}
	r := mux.NewRouter()
	r.Use(api.apiKeyMiddleware)
	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/status", api.Status)
	postOnly(v1.HandleFunc("/poll_block", api.PollBlock))
	getOnly(v1.HandleFunc("/accounts", api.HandleAccountsGET))
	postOnly(v1.HandleFunc("/accounts", api.HandleAccountsPOST))
	accounts := v1.PathPrefix("/accounts/{accountID}").Subrouter()
	getOnly(accounts.HandleFunc("/", api.HandleAccountGET))
	jsonPostOnly(accounts.HandleFunc("/unlock", api.HandleAccountUnlockPOST))
	jsonPostOnly(accounts.HandleFunc("/lock", api.HandleAccountLockPOST))
	getOnly(accounts.HandleFunc("/transactions", api.HandleAccountTransactionsGET))
	getOnly(accounts.HandleFunc("/coins", api.HandleCoinsGET))
	getOnly(accounts.HandleFunc("/names", api.HandleNamesGET))
	getOnly(accounts.HandleFunc("/unspent_bids", api.HandleUnspentBidsGET))
	getOnly(accounts.HandleFunc("/unspent_reveals", api.HandleUnspentRevealsGET))
	getOnly(accounts.HandleFunc("/names/{name}", api.HandleNameGET))
	jsonPostOnly(accounts.HandleFunc("/receive_address", api.HandleGenerateReceiveAddress))
	jsonPostOnly(accounts.HandleFunc("/change_address", api.HandleGenerateChangeAddress))
	jsonPostOnly(accounts.HandleFunc("/sends", api.HandleAccountSendPOST))
	jsonPostOnly(accounts.HandleFunc("/opens", api.HandleAccountOpensPOST))
	jsonPostOnly(accounts.HandleFunc("/bids", api.HandleAccountBidsPOST))
	jsonPostOnly(accounts.HandleFunc("/reveals", api.HandleAccountRevealsPOST))
	jsonPostOnly(accounts.HandleFunc("/redeems", api.HandleAccountRedeemsPOST))
	jsonPostOnly(accounts.HandleFunc("/updates", api.HandleAccountUpdatesPOST))
	jsonPostOnly(accounts.HandleFunc("/transfers", api.HandleAccountTransfersPOST))
	jsonPostOnly(accounts.HandleFunc("/finalizes", api.HandleAccountFinalizesPOST))
	jsonPostOnly(accounts.HandleFunc("/renewals", api.HandleAccountRenewalsPOST))
	jsonPostOnly(accounts.HandleFunc("/revokes", api.HandleAccountRevokesPOST))
	jsonPostOnly(accounts.HandleFunc("/dutch_auction_listing_transfers", api.HandleDutchAuctionListingTransfersPOST))
	jsonPostOnly(accounts.HandleFunc("/dutch_auction_listing_finalizes", api.HandleDutchAuctionListingFinalizesPOST))
	jsonPostOnly(accounts.HandleFunc("/dutch_auction_cancel_transfers", api.HandleDutchAuctionCancelTransfersPOST))
	jsonPostOnly(accounts.HandleFunc("/dutch_auction_cancel_finalizes", api.HandleDutchAuctionCancelFinalizesPOST))
	jsonPostOnly(accounts.HandleFunc("/dutch_auction_listings", api.HandleUpdateDutchAuctionListingsPOST))
	jsonPostOnly(accounts.HandleFunc("/dutch_auction_fill_transfers", api.HandleDutchAuctionFillTransfersPOST))
	jsonPostOnly(accounts.HandleFunc("/dutch_auction_fill_finalizes", api.HandleDutchAuctionFillFinalizesPOST))
	jsonPostOnly(accounts.HandleFunc("/zap", api.HandleZapPost))
	jsonPostOnly(accounts.HandleFunc("/rescan", api.HandleRescanPOST))
	jsonPostOnly(accounts.HandleFunc("/sign_message", api.HandleSignMessagePOST))
	jsonPostOnly(accounts.HandleFunc("/sign_message_with_name", api.HandleSignMessageWithNamePOST))
	return r
}

func (a *API) Status(w http.ResponseWriter, r *http.Request) {
	MarshalResponseJSON(w, a.node.Status())
}

func (a *API) PollBlock(w http.ResponseWriter, r *http.Request) {
	err := a.node.PollBlock()
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}
	w.WriteHeader(204)
}

func (a *API) apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		providedKey := r.Header.Get("X-API-Key")
		if providedKey != a.apiKey {
			MarshalErrorJSON(w, errors.New("invalid API key"), 401)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getOnly(route *mux.Route) {
	route.Methods("GET")
}

func postOnly(route *mux.Route) *mux.Route {
	route.Methods("POST")
	return route
}

func jsonPostOnly(route *mux.Route) {
	postOnly(route).
		Headers("Content-Type", "application/json")
}
