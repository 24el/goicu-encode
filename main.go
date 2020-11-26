package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type Options struct {
	FilePath string
}

func (o *Options) Validate() error {
	if o.FilePath == "" {
		return errors.New("file path required")
	}

	return nil
}

func main() {
	o := &Options{}

	cmd := &cobra.Command{
		Use: "goicu-encode",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Validate(); err != nil {
				return err
			}

			return o.encode()
		},
	}

	cmd.Flags().StringVarP(&o.FilePath, "file", "f", "", "go18n file format in json")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type PluralVal struct {
	PType string
	PVal  string
}

type PluralVals []*PluralVal

func (v PluralVals) Len() int {
	return len(v)
}

func orderPluralTypes(t string) int8 {
	if strings.Contains(t, "=") {
		return 1
	}

	switch t {
	case "one":
		return 2
	case "two":
		return 3
	case "few":
		return 4
	case "many":
		return 5
	case "other":
		return 6
	}

	return 0
}

func (v PluralVals) Less(i, j int) bool {
	io, jo := orderPluralTypes(v[i].PType), orderPluralTypes(v[j].PType)

	return io < jo
}

func (v PluralVals) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (o *Options) encode() error {
	f, err := os.OpenFile(o.FilePath, os.O_RDWR, 755)
	if err != nil {
		return err
	}

	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	translations := make(map[string]interface{})

	if err := json.Unmarshal(b, &translations); err != nil {
		return err
	}

	prepared := make(map[string]string)

	for k, message := range translations {
		switch t := message.(type) {
		case string:
			prepared[k] = strings.ReplaceAll(strings.ReplaceAll(t, "{{.", "{"), "}}", "}")
		case map[string]interface{}:
			b := strings.Builder{}
			b.WriteString("{PluralCount, plural,")

			vals := make(PluralVals, 0, len(t))

			for pt, pv := range t {
				pvs := pv.(string)

				pvs = strings.ReplaceAll(pvs, "{{.PluralCount}}", "#")

				pvs = strings.ReplaceAll(strings.ReplaceAll(pvs, "{{.", "{"), "}}", "}")

				vals = append(vals, &PluralVal{
					PType: strings.ToLower(pt),
					PVal:  pvs,
				})
			}

			sort.Sort(vals)

			for _, pv := range vals {
				b.WriteString(fmt.Sprintf(" %s {%s}", pv.PType, pv.PVal))
			}

			b.WriteString("}")

			prepared[k] = b.String()
		}
	}

	out, err := json.MarshalIndent(prepared, "", "  ")
	if err != nil {
		return err
	}

	if err := f.Truncate(0); err != nil {
		return err
	}

	if _, err := f.Seek(0, 0); err != nil {
		return err
	}

	_, err = f.Write(out)

	return err
}
