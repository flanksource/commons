# Decimal and Money

Import `github.com/flanksource/commons/decimal` for 19-digit half-even decimals backed by `github.com/govalues/decimal`. Import `github.com/flanksource/commons/bigdecimal` for higher precision backed by `github.com/woodsbury/decimal128`. Values are separate types; conversion is explicit. Neither package provides arbitrary precision.

Use `Parse(string)` for runtime input and `MustParse(string)` for constants. Arithmetic delegates to the upstream library and returns Commons values. Decimal arithmetic returns errors such as overflow and division by zero; bigdecimal preserves upstream special-value arithmetic. The bigdecimal `Decimal.Compare` method and the package-level `bigdecimal.Compare` both apply the upstream total order and never panic: NaN is less than every other value and equal to NaN. The embedded `.Decimal` field provides explicit access to upstream functionality.

`String()`, `%s`, `%v`, JSON, and text use canonical fixed-point notation without insignificant fractional zeros. `%f` accepts an explicit precision for rounding and padding. JSON always emits quoted numeric strings and accepts strings or numeric tokens without a float conversion. JSON rejects null, malformed input, and nonfinite values. Use pointers for nullable fields. Text decoding is atomic; bigdecimal text serialization rejects nonfinite values.

Both packages expose `NewMoney(MoneyOptions)`. `Amount` is a finite plain-decimal string parsed with the selected backend's rounding semantics. `Currency` must contain three uppercase letters; no currency registry or exchange-rate lookup is performed. `DecimalPlaces` is the minimum display scale from 0 to 31, not an amount rounding instruction. Money strings and JSON include the currency, such as `USD 125.00`.

Money exposes `Amount()`, `Currency()`, `DecimalPlaces()`, and `Cmp(other)`. Comparisons require initialized values and the same currency. Money arithmetic and implicit precision/currency conversion are not provided. Construct Money with `NewMoney`; uninitialized values fail validation and serialization, and String formats them as `<uninitialized Money>`.

Run the examples and regression suite from the Commons root:

```sh
go test ./decimal ./bigdecimal
```

Regenerate numeric delegates with the pinned upstream modules downloaded by `go mod download`. Pass the corresponding module-cache directory to the generator:

```sh
go run ./internal/gendecimal decimal /path/to/github.com/govalues/decimal@v0.1.36 github.com/govalues/decimal
go run ./internal/gendecimal bigdecimal /path/to/github.com/woodsbury/decimal128@v1.4.1 github.com/woodsbury/decimal128
```
