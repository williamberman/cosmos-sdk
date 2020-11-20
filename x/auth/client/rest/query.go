package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	genutilrest "github.com/cosmos/cosmos-sdk/x/genutil/client/rest"
)

// query accountREST Handler
func QueryAccountRequestHandlerFn(storeName string, clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bech32addr := vars["address"]

		addr, err := sdk.AccAddressFromBech32(bech32addr)
		if rest.CheckInternalServerError(w, err) {
			return
		}

		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		accGetter := types.AccountRetriever{}

		account, height, err := accGetter.GetAccountWithHeight(clientCtx, addr)
		if err != nil {
			// TODO: Handle more appropriately based on the error type.
			// Ref: https://github.com/cosmos/cosmos-sdk/issues/4923
			if err := accGetter.EnsureExists(clientCtx, addr); err != nil {
				clientCtx = clientCtx.WithHeight(height)
				rest.PostProcessResponse(w, clientCtx, types.BaseAccount{})
				return
			}

			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		clientCtx = clientCtx.WithHeight(height)
		rest.PostProcessResponse(w, clientCtx, account)
	}
}

// QueryTxsRequestHandlerFn implements a REST handler that searches for transactions.
// Genesis transactions are returned if the height parameter is set to zero,
// otherwise the transactions are searched for by events.
func QueryTxsRequestHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			rest.WriteErrorResponse(
				w, http.StatusBadRequest,
				fmt.Sprintf("failed to parse query parameters: %s", err),
			)
			return
		}

		// if the height query param is set to zero, query for genesis transactions
		heightStr := r.FormValue("height")
		if heightStr != "" {
			if height, err := strconv.ParseInt(heightStr, 10, 64); err == nil && height == 0 {
				genutilrest.QueryGenesisTxs(clientCtx, w)
				return
			}
		}

		var (
			events      []string
			txs         []sdk.TxResponse
			page, limit int
		)

		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		if len(r.Form) == 0 {
			rest.PostProcessResponseBare(w, clientCtx, txs)
			return
		}

		events, page, limit, err = rest.ParseHTTPArgs(r)
		if rest.CheckBadRequestError(w, err) {
			return
		}

		searchResult, err := authclient.QueryTxsByEvents(clientCtx, events, page, limit, "")
		if rest.CheckInternalServerError(w, err) {
			return
		}

		for _, txRes := range searchResult.Txs {
			packStdTxResponse(w, clientCtx, txRes)
		}

		rest.PostProcessResponseBare(w, clientCtx, searchResult)
	}
}

// QueryTxRequestHandlerFn implements a REST handler that queries a transaction
// by hash in a committed block.
func QueryTxRequestHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hashHexStr := vars["hash"]

		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		output, err := authclient.QueryTx(clientCtx, hashHexStr)
		if err != nil {
			if strings.Contains(err.Error(), hashHexStr) {
				rest.WriteErrorResponse(w, http.StatusNotFound, err.Error())
				return
			}
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		err = packStdTxResponse(w, clientCtx, output)
		if err != nil {
			// Error is already returned by packStdTxResponse.
			return
		}

		if output.Empty() {
			rest.WriteErrorResponse(w, http.StatusNotFound, fmt.Sprintf("no transaction found with hash %s", hashHexStr))
		}

		rest.PostProcessResponseBare(w, clientCtx, output)
	}
}

func queryParamsHandler(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryParams)
		res, height, err := clientCtx.QueryWithData(route, nil)
		if rest.CheckInternalServerError(w, err) {
			return
		}

		clientCtx = clientCtx.WithHeight(height)
		rest.PostProcessResponse(w, clientCtx, res)
	}
}

// packStdTxResponse takes a sdk.TxResponse, converts the Tx into a StdTx, and
// packs the StdTx again into the sdk.TxResponse Any. Amino then takes care of
// seamlessly JSON-outputting the Any.
func packStdTxResponse(w http.ResponseWriter, clientCtx client.Context, txRes *sdk.TxResponse) error {
	// We just unmarshalled from Tendermint, we take the proto Tx's raw
	// bytes, and convert them into a StdTx to be displayed.
	txBytes := txRes.Tx.Value
	fmt.Println("*******************")
	fmt.Println("Before convert")
	fmt.Println(txBytes)
	fmt.Println("*******************")
	stdTx, err := convertToStdTx(w, clientCtx, txBytes)
	fmt.Println("*******************")
	fmt.Println("after convert")
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	fmt.Println("*******************")
	
	fmt.Println("*******************")
	fmt.Println("Before pack")
	fmt.Println(stdTx)
	fmt.Println("*******************")

	// Pack the amino stdTx into the TxResponse's Any.
	txRes.Tx = codectypes.UnsafePackAny(stdTx)

	
	fmt.Println("*******************")
	fmt.Println("After pack")
	fmt.Println(txRes.Tx)
	fmt.Println("*******************")

	return nil
}
