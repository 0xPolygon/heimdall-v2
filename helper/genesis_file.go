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

// TODO HV2 Once the genesis files are ready, please add them inside the alloc folder and
// uncomment the following "go:embed allocs" comment.

/*
//go:embed allocs
*/

var allocs embed.FS

func WriteGenesisFile(chain string, filePath string) error {
	switch chain {
	case "amoy", "mumbai", "mainnet":
		fn := fmt.Sprintf("allocs/%s.json", chain)

		genDoc, err := readPrealloc(fn)
		if err == nil {
			err = genDoc.SaveAs(filePath)
		}

		return err
	default:
		return errors.New("invalid chain name")
	}
}

func readPrealloc(filename string) (result cmTypes.GenesisDoc, err error) {
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
		err = codec.NewLegacyAmino().UnmarshalJSON(buf.Bytes(), &result)
	}

	return
}
