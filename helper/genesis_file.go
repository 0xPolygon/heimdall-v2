package helper

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"

	cmTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/pkg/errors"
)

//go:embed allocs
var allocs embed.FS

func WriteGenesisFile(chain string, filePath string, cdc *codec.Codec) error {
	switch chain {
	case "amoy", "mumbai", "mainnet":
		fn := fmt.Sprintf("allocs/%s.json", chain)

		genDoc, err := readPrealloc(fn, cdc)
		if err == nil {
			err = genDoc.SaveAs(filePath)
		}

		return err
	default:
		return nil
	}
}

func readPrealloc(filename string, cdc *codec.Codec) (result cmTypes.GenesisDoc, err error) {
	f, err := allocs.Open(filename)
	if err != nil {
		err = errors.Errorf("could not open genesis preallocation for %s: %v", filename, err)
		return
	}
	defer func(f fs.File) {
		err := f.Close()
		if err != nil {
			Logger.Error("error while closing file handler: %v", err)
		}
	}(f)

	buf := bytes.NewBuffer(nil)

	_, err = buf.ReadFrom(f)
	if err == nil {
		err = (*cdc).UnmarshalInterfaceJSON(buf.Bytes(), &result)
	}

	return
}
