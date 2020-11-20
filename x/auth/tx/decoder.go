package tx

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

// DefaultTxDecoder returns a default protobuf TxDecoder using the provided Marshaler.
func DefaultTxDecoder(cdc codec.ProtoCodecMarshaler) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		fmt.Println("***************")
		fmt.Println("x/auth/tx/decoder.go 1")
		fmt.Println("***************")
		var raw tx.TxRaw

		// reject all unknown proto fields in the root TxRaw
		err := unknownproto.RejectUnknownFieldsStrict(txBytes, &raw, cdc.InterfaceRegistry())
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, err.Error())
		}

		err = cdc.UnmarshalBinaryBare(txBytes, &raw)
		if err != nil {
			return nil, err
		}

		var body tx.TxBody

		fmt.Println("***************")
		fmt.Println("x/auth/tx/decoder.go string(raw.BodyBytes)")
		fmt.Println(len(raw.BodyBytes))
		fmt.Println(string(raw.BodyBytes))
		fmt.Println("***************")

		// allow non-critical unknown fields in TxBody
		txBodyHasUnknownNonCriticals, err := unknownproto.RejectUnknownFields(raw.BodyBytes, &body, true, cdc.InterfaceRegistry())
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, err.Error())
		}
		
		fmt.Println("***************")
		fmt.Println("x/auth/tx/decoder.go string(raw.BodyBytes) 1")
		fmt.Println(len(raw.BodyBytes))
		fmt.Println(txBodyHasUnknownNonCriticals)
		fmt.Println(string(raw.BodyBytes))
		fmt.Println("***************")

		err = cdc.UnmarshalBinaryBare(raw.BodyBytes, &body)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, err.Error())
		}

		var authInfo tx.AuthInfo

		// reject all unknown proto fields in AuthInfo
		err = unknownproto.RejectUnknownFieldsStrict(raw.AuthInfoBytes, &authInfo, cdc.InterfaceRegistry())
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, err.Error())
		}

		err = cdc.UnmarshalBinaryBare(raw.AuthInfoBytes, &authInfo)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, err.Error())
		}

		theTx := &tx.Tx{
			Body:       &body,
			AuthInfo:   &authInfo,
			Signatures: raw.Signatures,
		}

		rv := &wrapper{
			tx:                           theTx,
			bodyBz:                       raw.BodyBytes,
			authInfoBz:                   raw.AuthInfoBytes,
			txBodyHasUnknownNonCriticals: txBodyHasUnknownNonCriticals,
		}

		fmt.Println("***************")
		fmt.Println("x/auth/tx/decoder.go rv.tx.GetMsgs()")
		fmt.Println(rv)
		fmt.Println(rv.tx.GetMsgs())
		fmt.Println("***************")

		return rv, nil
	}
}

// DefaultJSONTxDecoder returns a default protobuf JSON TxDecoder using the provided Marshaler.
func DefaultJSONTxDecoder(cdc codec.ProtoCodecMarshaler) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		var theTx tx.Tx
		err := cdc.UnmarshalJSON(txBytes, &theTx)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, err.Error())
		}

		return &wrapper{
			tx: &theTx,
		}, nil
	}
}
