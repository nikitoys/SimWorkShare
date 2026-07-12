# SimWorkShare

Детерминированная реализация сравнения фиксированной оплаты и ежемесячного
profit sharing из спецификации v0.3.

Текущий срез последовательно выполняет `simulation.months` месяцев для:

- `fixed_only + no_effect`;
- monthly `profit_share` с явно выбранным behavior case;
- `normal_market` с нейтральными детерминированными факторами.

Между месяцами переносятся cash, accounts receivable и tax payable. AR и налог
исполняются через отдельные очереди со сроками из конфигурации. Profit-share
bonus accrual, payroll tax, restricted bonus cash и bonus payable queue
реализованы для monthly period и payout lag `>= 1`. Monte Carlo, quarterly и
annual bonus, fixed-raise prepass, sensitivity analysis, classification и
полный набор отчётов намеренно ещё не реализованы.

## Запуск

Из корня репозитория:

```powershell
go run ./cmd/profitshare-sim -config ./doc/default_config_v0_3_implementation_ready.json
```

Команда печатает в stdout стабильный JSON с массивом `monthly_results` и
`terminal_summary`. Для canonical config массив содержит 60 месяцев. Повторный
запуск с тем же config даёт побайтово тот же результат; `simulation.runs`, seed
и common random numbers на этом детерминированном этапе не используются.

Выбрать monthly profit sharing:

```powershell
go run ./cmd/profitshare-sim -config ./doc/default_config_v0_3_implementation_ready.json -scenario profit_share_equal_10 -behavior no_effect
```

Сразу сравнить фиксированную схему и 10% profit sharing на общей экономике:

```powershell
go run ./cmd/profitshare-compare -config ./doc/default_config_v0_3_implementation_ready.json -profit-scenario profit_share_equal_10 -profit-behavior no_effect
```

Для проверки поведенческой гипотезы `-profit-behavior` можно явно заменить на
`moderate_effect` или `optimistic_effect`. Это сценарные допущения, а не эффект,
который модель выводит автоматически.

## Два профиля сравнения

Готовы два тонких профиля, которые используют один и тот же base config и
различаются только схемой оплаты:

```powershell
go run ./cmd/profitshare-sim -profile ./profiles/fixed_only.json
go run ./cmd/profitshare-sim -profile ./profiles/profit_share_10.json
```

Оба профиля помечены `template_defaults_not_real_data`: canonical числа пока
не являются фактическими показателями компании. После ввода и ручной проверки
реальных P&L/cash/workforce параметров в общем `base_config` статус профилей
можно сменить на `calibrated`; сам флаг не подтверждает качество калибровки.

## Проверка

```powershell
go test ./...
go vet ./...
```

`baseline_fixed_only_sanity` из раздела 18 проверяется отдельным fixture, в
котором прямые turnover costs равны нулю. Полный default config сохраняет
базовую текучесть и поэтому включает её стоимость в operating profit. Причина
этого разделения записана в [DECISIONS.md](./DECISIONS.md).

Дополнительно тесты проверяют точное сохранение результата первого месяца,
непрерывность opening/closing cash, балансы AR и tax payable, однократную оплату
обязательств в due month, отсутствие bonus state и повторяемость 60-месячного
запуска, а также bonus accrual/payment ledgers, payroll tax, cash affordability,
caps, hurdle и парный cash bridge между схемами.

## Порядок месяца и очереди

В каждом месяце сначала рассчитываются environment, workforce и operating P&L,
затем собирается наступившая AR и оплачиваются наступившие обязательства,
включая старый bonus due. Затем новый бонус начисляется в P&L, его employer cost
резервируется без уменьшения total cash, и только после этого начисляется налог
на прибыль. Налог и бонус уменьшают cash только в due month.

Opening AR считается уже очищенной от bad debt и собирается в месяце 1. Новая
AR после bad-debt haircut и текущий tax accrual получают срок
`origin_month + configured_lag`. Хвосты очередей за пределами горизонта не
форсируются к оплате и остаются в terminal summary.

## Денежные величины

Деньги хранятся как `float64` в номинальных единицах
`simulation.currency`. Промежуточное округление не выполняется. Бухгалтерские
равенства используют централизованный absolute/relative tolerance. Пороговые
risk-сравнения используют только абсолютный допуск `1e-6`, чтобы tolerance не
рос вместе с балансом. Форматирование и округление относятся только к будущему
слою представления.

## Структура

- `internal/config` — строгая загрузка, нормализация и валидация JSON;
- `internal/domain` — типы состояния и результата;
- `internal/model` — чистые расчёты workforce, P&L и cash;
- `internal/sim` — многомесячная оркестрация, due queues и paired comparison;
- `internal/runprofile` — строгая загрузка тонких профилей запуска;
- `cmd/profitshare-sim` — минимальный CLI;
- `cmd/profitshare-compare` — сравнение fixed_only и profit_share одним запуском.

Исходная спецификация и canonical default config остаются в каталоге `doc/`.
