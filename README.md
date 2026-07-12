# SimWorkShare v0.4

SimWorkShare — месячная имитационная модель для сравнения четырёх
организационных сценариев:

- `traditional_company`;
- `profit_sharing`;
- `employee_ownership_partial`;
- `worker_cooperative`.

Модель разделяет ownership, employee cash distribution, member capital,
governance, reinvestment и external capital. Она считает динамический
headcount, ramp-up cohorts, производственные ограничения, P&L, cash и debt,
денежные очереди, capacity, риски и метрики развития. Это сценарный инструмент,
а не доказательство преимуществ какой-либо формы организации и не оценка доли
основателя.

## Что реализовано

- строгая загрузка полного JSON v0.4: неизвестные и дублирующиеся поля,
  пропуски, `NaN`/`Infinity`, enum, диапазоны и межполевые правила отклоняются с
  полным путём к полю;
- deterministic и Monte Carlo, stable seed, независимые random streams и common
  random numbers;
- точный месячный pipeline разделов 7–8, пять due queues и инварианты раздела
  18;
- четыре организационных сценария со всеми разрешёнными behavior cases,
  включая `no_effect` и отрицательные controls;
- monthly results, per-run и aggregate terminal summaries, P10/median/P90,
  paired deltas к `traditional_company` и `profit_sharing`, risk/workforce/
  employee/development metrics;
- sensitivity grid и break-even productivity uplift с risk gates;
- JSON, monthly CSV, summary CSV, paired CSV, sensitivity CSV и break-even CSV;
- отдельный compatibility path для реально работавшего v0.3 без изменения
  старых golden fixtures.

Канонический `doc/default_config_v0_4.json` — Monte Carlo на 1000 runs,
240 месяцев с горизонтами 60/120/240 и шаблонными, не откалиброванными данными.
Для быстрой проверки используйте меньшее `-runs`; для итогового анализа уберите
override.

## Требования

Модуль объявляет Go 1.26. Если `go` отсутствует в `PATH`, на Windows можно
заменить его в командах на `C:\Program Files\Go\bin\go.exe`.

## Запуск v0.4

Подготовьте каталог для файлов результатов:

```powershell
New-Item -ItemType Directory -Force ./out
```

Детерминированный сценарий с полными месячными и terminal outputs:

```powershell
go run ./cmd/profitshare-sim `
  -config ./doc/default_config_v0_4.json `
  -mode deterministic `
  -scenario worker_cooperative `
  -behavior no_effect `
  -monthly-csv ./out/monthly.csv `
  -summary-csv ./out/summary.csv `
  -output ./out/result.json
```

Monte Carlo с тем же seed воспроизводится побитово в рамках реализации stable
RNG. Команда ниже — короткий аналитический запуск на 100 runs:

```powershell
go run ./cmd/profitshare-sim `
  -config ./doc/default_config_v0_4.json `
  -scenario worker_cooperative `
  -behavior moderate_positive `
  -runs 100 `
  -seed 20260712 `
  -summary-csv ./out/monte_carlo_summary.csv `
  -output ./out/monte_carlo.json
```

Paired comparison автоматически добавляет оба reference-сценария из config,
даже если выбран один candidate:

```powershell
go run ./cmd/profitshare-sim `
  -config ./doc/default_config_v0_4.json `
  -scenario worker_cooperative `
  -behavior moderate_positive `
  -runs 100 `
  -paired-csv ./out/paired.csv `
  -output ./out/comparison.json
```

Sensitivity grid:

```powershell
go run ./cmd/profitshare-sim `
  -config ./doc/default_config_v0_4.json `
  -scenario worker_cooperative `
  -behavior moderate_positive `
  -runs 20 `
  -sensitivity `
  -sensitivity-csv ./out/sensitivity.csv `
  -output ./out/sensitivity.json
```

Break-even относительно traditional company:

```powershell
go run ./cmd/profitshare-sim `
  -config ./doc/default_config_v0_4.json `
  -scenario worker_cooperative `
  -behavior moderate_positive `
  -reference traditional_company `
  -horizon 120 `
  -runs 20 `
  -break-even `
  -break-even-csv ./out/break_even.csv `
  -output ./out/break_even.json
```

Без `-output` JSON печатается в stdout. Предупреждения и ошибки идут только в
stderr. Monte Carlo monthly rows включаются в JSON флагом
`-include-monthly-json`, per-run summaries — `-include-run-summaries`. CSV-файл
создаётся только при переданном пути; `reporting.write_*` в config должен
разрешать соответствующий формат.

`shock_survival_rate` рассчитывается только для шоков с полным 12-месячным
окном наблюдения. Если таких прогонов нет, JSON содержит `null`, а соответствующая
ячейка summary CSV остаётся пустой.

## Совместимость v0.3

Старые команды сохранены:

```powershell
go run ./cmd/profitshare-sim -config ./doc/default_config_v0_3_implementation_ready.json
go run ./cmd/profitshare-sim -config ./doc/default_config_v0_3_implementation_ready.json -scenario profit_share_equal_10 -behavior no_effect
go run ./cmd/profitshare-compare -config ./doc/default_config_v0_3_implementation_ready.json -profit-scenario profit_share_equal_10 -profit-behavior no_effect
```

Compatibility охватывает только действительно исполнявшиеся v0.3
`fixed_only` и monthly profit-sharing scenarios. Fixed raise, quarterly и
annual definitions старый код не реализовывал; их выбор возвращает явную
unsupported-feature error.

## Проверка

```powershell
Get-ChildItem ./cmd,./internal -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
go test ./...
go vet ./...
go test -race ./...
```

Race detector требует рабочий CGO toolchain. Если он недоступен, обычные tests
и vet по-прежнему выполняются, а ограничение race-прогона следует фиксировать
отдельно.

## Структура

- `internal/v04/config` — отдельные v0.4 types, strict parser, validation,
  warnings и sensitivity mutation;
- `internal/v04/domain` — monthly state, summaries и output contracts;
- `internal/v04/sim` — environment, workforce, behavior, governance,
  production, economics, allocation, financing, queues, member capital,
  metrics, sensitivity, break-even и invariants;
- `internal/compatv03` и прежние `internal/*` — воспроизводимость v0.3;
- `cmd/profitshare-sim` — v0.4 runner и v0.3 compatibility wrapper;
- `cmd/profitshare-compare` — сохранённое v0.3 comparison CLI;
- `doc/` — спецификация, canonical configs, schema, test manifest и migration
  artifacts.

Все неоднозначности формул и границы совместимости зафиксированы в
[DECISIONS.md](./DECISIONS.md).

## Ограничения

Tax, external capital, governance и member capital являются упрощёнными
механизмами, а workforce агрегирован. High-performer inputs выводятся как
assumption indicator, но не входят в aggregate turnover: раздел 8 не задаёт
для них формулу. `sustainable_development_value_proxy` не является equity или
founder value. Любой вывод следует проверять на sensitivity и калиброванных
данных.
