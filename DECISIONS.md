# Публичные решения модели — детерминированные этапы 1–3

Этот файл фиксирует трактовки, необходимые для текущей исполняемой версии.
Решения относятся к `fixed_only` и monthly `profit_share` в deterministic
`normal_market` и не определяют будущую реализацию Monte Carlo.

## D-001. Единицы денег и числовые допуски

- Все денежные значения хранятся как `float64` в номинальных единицах
  `simulation.currency`.
- Recurring денежные входы считаются месячными, если явно не указано иное.
- Промежуточного округления нет.
- Сравнение денег использует:
  `abs(a-b) <= max(1e-6, 1e-9 * max(1, abs(a), abs(b)))`.
- Для ledger identity вида `closing = opening - paid + accrued` relative scale
  включает все четыре операнда. Это защищает проверку от catastrophic
  cancellation, когда большой due скрывает малый будущий queue entry в
  агрегированном `float64`; authoritative closing balance берётся из очереди.
- Проверки policy/risk thresholds используют строгий `<` с единственным
  абсолютным защитным допуском `1e-6`; relative tolerance к ним не применяется.
- Округление является обязанностью слоя представления, которого на этом этапе
  ещё нет.

Причина: deterministic expected leavers могут быть дробными, поэтому integer
minor units создавали бы ложное округление внутри модели.

## D-002. Canonical config и defaults

Файл `doc/default_config_v0_3_implementation_ready.json` считается полным
canonical default, а не частичным overlay. Обязательные поля нельзя молча
восстанавливать из нулевых значений Go.

Нормализуются только явно определённые type-specific defaults profit-share
policy: hurdle `0`, base type `distributable_base`, eligible employees равны
`company.employees_count`, caps равны `null`, period `monthly`, payout lag `1`,
equal distribution `true`, smoothing reserve rate `0`. Эти definitions
загружаются и валидируются. Monthly definitions исполняются при явном выборе;
quarterly/annual пока остаются только валидируемыми definitions.

## D-003. Месяцы и normal_market

- Первый исполняемый месяц имеет индекс `1`.
- Cumulative market trend и cost inflation используют степень `month-1`, поэтому
  в первом месяце оба фактора равны `1`, а далее накапливаются геометрически.
- Детерминированный `normal_market` использует `market_factor=1`, отсутствие
  shock и `collection_rate_multiplier=1`.
- Волатильность и вероятность shock не заменяются expected values: они просто
  не исполняются до этапа Monte Carlo.

## D-004. Headcount и turnover

- Headcount в v1 постоянный; ушедшие сотрудники считаются мгновенно заменёнными.
- В `deterministic` mode leavers — дробное ожидаемое количество.
- Базовая текучесть сохраняется даже для `no_effect` и создаёт turnover cost.

## D-005. Sanity test из раздела 18

Ожидаемый operating profit `1 625 000` не включает turnover cost, хотя default
config содержит ненулевую текучесть и costs per leaver. Поэтому
`baseline_fixed_only_sanity` является отдельным company-economics fixture с
нулевыми денежными costs per leaver.

Полный запуск default config не исключает turnover ради совпадения с sanity
числом.

## D-006. Accounts receivable queue

Вся `opening_accounts_receivable`:

- считается уже очищенной от bad debt;
- собирается в первом месяце;
- не смешивается с новой AR текущего месяца.

Новая AR уже учитывает bad-debt haircut и считается полностью собираемой в
месяце `origin_month + accounts_receivable_lag_months`. На промежуточных этапах
очередь не округляется и не отбрасывает малые ненулевые суммы. После сбора due
entry удаляется, поэтому повторно собран быть не может.

Баланс каждого месяца:
`AR_closing = AR_opening - AR_collected + AR_new`. Closing AR текущего месяца
становится opening AR следующего месяца. Хвост с due month за пределами
simulation horizon остаётся outstanding, а не собирается досрочно.

## D-007. Accounting и cash

- Salary, fixed, variable, turnover, shock, debt service и CAPEX при отсутствии
  отдельного лага оплачиваются в месяце начисления.
- Profit tax начисляется в текущем P&L, но при default lag `1` не уменьшает cash
  первого месяца и сохраняется как closing tax payable.
- Opening tax payable, bonus payable и restricted bonus cash считаются нулевыми:
  canonical config не содержит входов для начальных значений этих очередей.
- Текущий execution profile поддерживает `profit_tax_payment_lag_months >= 1`.
  Налог оплачивается ровно один раз в месяце
  `origin_month + profit_tax_payment_lag_months`; due entry после оплаты
  удаляется.
- Баланс каждого месяца:
  `tax_payable_closing = tax_payable_opening - tax_paid + tax_accrual`.
  Налоговый хвост за горизонтом остаётся closing liability.
- Tax payable не является restricted bonus cash и до фактической оплаты не
  уменьшает cash. На этом этапе он также не вычитается из расчётного поля
  owner distributable cash.
- P&L state и cash state являются отдельными типами.
- Debt service, CAPEX и cash tax payments не входят в operating P&L. Текущий
  tax accrual не входит в cash payments текущего месяца.
- Profit-share bonus и bonus payroll tax входят в P&L только в месяц начисления.
  Их последующая выплата является только cash settlement и второй раз расходом
  не признаётся.

## D-008. Граница текущего исполнения

Config может содержать определения будущих compensation, behavior и environment
cases. Наличие этих definitions не является ошибкой. Fixed-only arm исполняется
только с `no_effect`; monthly profit-share arm использует явно выбранный
declared behavior case. Среда текущего этапа — только `normal_market`.

Исполняются все месяцы от `1` до `simulation.months`. Closing cash текущего
месяца становится opening cash следующего без округления. Bankruptcy является
флагом риска и не останавливает расчёт заданного горизонта. Available credit
line задаёт только порог bankruptcy и не создаёт денежного притока.

Исполняются `fixed_only` и monthly `profit_share` с payout lag `>= 1`.

Не исполняются:

- `simulation.runs`, RNG и common random numbers;
- quarterly/annual profit sharing и same-month bonus payout (`lag 0`);
- fixed raise и prepass;
- автоматический вывод behavior effect из размера или факта выплаты бонуса;
- environment cases кроме `normal_market`;
- non-`none` owner dividends;
- sensitivity, aggregation и полный reporting.

Неподдерживаемые future definitions из canonical config разрешено загружать и
валидировать, но они не выбираются runner. Напротив, execution settings, для
которых текущий профиль не может дать корректный результат, отклоняются явно:
binomial turnover, non-`none` dividends и same-month profit-tax settlement
(`tax lag 0`).

CLI возвращает один `SimulationResult` с `monthly_results` и кратким
`terminal_summary`. Summary выводится только из рассчитанных месяцев и
не инициирует дополнительного settlement хвостов очередей.

## D-009. Profit-share formula и фиксированная часть

- В текущей схеме profit sharing не уменьшает фиксированный оклад. Salary costs
  рассчитываются так же, как в `fixed_only`; бонус начисляется сверху.
- Спецификация содержит конфликт: обзорная формула предлагает
  `percent * min(profit_base, cash_base)`, а детальная §9.7 ограничивает gross
  bonus непосредственно через cash affordability.
- Авторитетной принята §9.7:
  `gross = min(percent * period_profit_base, cash_base / (1 + payroll_tax), caps)`.
- Отчётный `distributable_base` равен `gross / percent` при ненулевом проценте.
- Total и per-employee caps ограничивают gross employee bonus, а не полный
  employer cost.
- Current tax reserve для affordability считается консервативно от
  pre-bonus operating profit; фактический profit tax начисляется после вычета
  gross bonus и bonus payroll tax.
- Planned reinvestment reserve ограничивает affordability, но не является cash
  outflow; фактический outflow по-прежнему отражается только через CAPEX.

## D-010. Bonus payable queue и restricted cash

- Opening bonus payable и opening restricted bonus cash равны нулю: входов для
  иных начальных значений в config нет.
- При начислении total cash не меняется; restricted bonus cash увеличивается на
  gross bonus плюс payroll tax.
- Выплата в due month уменьшает total cash и restricted cash на одинаковый
  employer cost и не меняет P&L повторно.
- Gross и payroll-tax части очереди хранятся и проверяются отдельно. Closing
  restricted cash обязан равняться полному closing bonus payable.
- Due entry удаляется после выплаты; хвост за горизонтом остаётся liability.
- Lag `0` пока отклоняется явно: месячный порядок спецификации обрабатывает due
  payments раньше нового accrual, а same-month settlement требует отдельного
  публичного решения.

## D-011. Честное сравнение и калибровка

- Fixed-only arm всегда использует `no_effect`. Для profit-share behavior case
  выбирается явно; модель не выводит productivity/retention effect из размера
  бонуса.
- `no_effect` показывает чистую стоимость схемы. `moderate_effect` и
  `optimistic_effect` являются проверяемыми сценарными предположениями.
- Два run profile используют один base config, чтобы экономика не расходилась
  между вариантами случайно.
- Текущие canonical значения являются шаблонными defaults, а не реальными
  показателями компании. До замены фактическими данными профили имеют статус
  `template_defaults_not_real_data`.

# Решения реализации v0.4

Секция ниже относится только к новому движку `internal/v04`. Исторические
решения D-001–D-011 сохранены как описание реально реализованного v0.3.

## D-040. Граница совместимости v0.3

- Канонический legacy-артефакт остаётся в `doc/legacy_v0_3/`. Копия по прежнему
  пути `doc/default_config_v0_3_implementation_ready.json` восстановлена для
  старых CLI, профилей и тестов.
- Compatibility-движок воспроизводит только фактически работавшие сценарии:
  `fixed_only` и ежемесячный profit sharing, включая существующие 5/10/15% и
  hurdle-варианты. Golden-файлы v0.3 не изменяются.
- Fixed-raise, quarterly и annual definitions присутствовали в конфигурации,
  но не исполнялись кодом v0.3. Они возвращают типизированную понятную ошибку,
  а не объявляются совместимыми задним числом.
- Переключатели `compatibility_v0_3.enabled`, `legacy_outputs_enabled` и
  `headcount_policy=fixed_v0_3` отклоняются валидатором v0.4. Legacy-расчёт
  запускается старой конфигурацией через сохранённые CLI, поэтому ни один
  валидный переключатель не игнорируется молча.
- `preserve_v0_3_profit_sharing_scenarios=true` в canonical config является
  декларативной metadata при `enabled=false`; включить compatibility через этот
  v0.4 engine всё равно нельзя. Фактические definitions сохраняются legacy
  config и compatibility fixtures.

## D-041. Единственный market case текущей схемы

В конфигурации v0.4 нет массива `market_cases`, хотя поле входит в архитектуру
и выходные форматы. Все результаты получают стабильный идентификатор
`default_market`. Он не зависит от порядка сценариев и не подразумевает
скрытого дополнительного рыночного сценария.

## D-042. Недоопределённые величины раздела 8

Приняты минимальные детерминированные трактовки без скрытых прогнозов:

- `market_demand_forecast_t = market_demand_t`, рассчитанный по текущему
  environment path, seasonality, shock response и governance до решения о
  найме. Отдельной прогнозной модели в конфигурации нет.
- `tax_reserve_estimate_t = max(0,
  profit_before_tax_before_distribution_t) * profit_tax_rate`. Это
  консервативный резерв до фактического вычета employee distribution.
- `unrestricted_cash_before_allocations_t = cash_total_after_mandatory_t -
  restricted_distribution_cash_after_due_t -
  restricted_reserve_cash_after_release_t`. Collections, reserve release и
  liquidity credit уже отражены; обязательные выплаты уже вычтены.
- Voluntary leavers и layoffs распределяются пропорционально между
  full-productivity headcount и всеми ramp cohorts. В integer-mode используется
  стабильный largest-remainder tie-break по индексу cohort.
- `income_volatility_index_12m_t` — population coefficient of variation по
  доступным предыдущим (не текущему) максимум 12 значениям per-employee
  income; при числе наблюдений меньше 2 или среднем `<= epsilon` результат 0,
  затем применяется `clamp(..., 0, 1)`.
- `member_capital_redemption_accrual_t = member_capital_begin_t *
  min(1, exits_t / max(epsilon, headcount_begin_t)) *
  member_capital_redemption_fraction_on_exit`. Начисление ограничено opening
  member capital и помещается в очередь на `t + configured_lag`.
- `net_profit_after_tax_and_employee_distribution_t =
  profit_before_tax_before_distribution_t - profit_tax_accrual_t -
  employee_distribution_gross_t - distribution_payroll_tax_t`.
  Reinvestment, reserve и member-capital allocation — использование результата,
  а не P&L expense.
- `pay_dispersion_index_t = 0`, поскольку конфигурируемого источника нет и это
  прямо предусмотренный спецификацией default.

## D-043. Начальные состояния, очереди и периодичность

- Opening debt, restricted distribution cash, restricted reserve, member
  capital, tax payable, distribution payable, redemption payable и capacity
  queue равны нулю: схема не содержит входов для иных значений. Единственное
  явное opening queue value — `opening_accounts_receivable`, due в месяц 1.
- `market_trend_0=1`; поэтому month 1 использует `(1 + market_growth_monthly)`.
  Cost inflation month 1 равен 1 и далее растёт степенью `month-1`.
- AR, tax, employee distribution, member redemption и capacity additions имеют
  отдельные due queues; хвост за горизонтом не форсируется к оплате.
- Для `profit_distribution_period_months > 1` накапливается сумма уже
  hurdled месячных `positive_result_base_t`. Employee cash distribution
  начисляется при закрытии периода; member capital, reinvestment, reserve и
  external distribution остаются месячными, как в формулах 8.11.
- Member-capital allocation не является текущим cash outflow, но имеет cash
  multiplier 1 и поэтому потребляет cash-safe allocation budget. Redemption —
  отдельный будущий денежный отток.
- `salary_costs_reference_monthly_t` для required reserve включает salary и
  employer salary payroll tax; иначе reserve систематически не покрывал бы
  обязательную полную стоимость payroll.
- External growth capital сразу направляется в investment: inflow и
  `external_capital_spent` равны; debt-вариант одновременно увеличивает debt.
- Stress funding сначала реклассифицирует не более configured share
  organizational reserve, затем использует доступный credit headroom.

## D-044. No-effect и отсутствующая high-performer формула

- Behavior case с точным именем `no_effect` валидируется семантически: все
  коэффициенты, способные создать productivity/retention effect, должны быть
  нулевыми. Governance hours, cash cost, delay и quality сохраняются как
  структурные реальные издержки.
- Спецификация содержит `high_performer_share` и
  `high_performer_attrition_delta_pp`, но не включает их в точную turnover
  формулу раздела 8.7. Чтобы не менять приоритетную формулу, эти поля не
  добавляются в aggregate turnover. Для ненулевых cases выводится явный
  assumption flag с weighted indicator; ограничение также печатается в списке
  model limitations.

## D-045. Метрики и анализ

- `average_employee_income_monthly = sum(actual salary paid + employee cash
  distribution paid) / sum(paid_employees_t)`. Это эквивалентно average monthly
  payments, делённым на average paid headcount, и не даёт месяцам с малым
  headcount непропорциональный вес.
- `employee_income_volatility` — population standard deviation месячного
  per-employee income; risk-adjusted income использует именно её.
- `actual_reinvestment_total = cash_reinvestment_paid +
  external_growth_capital_draw`; underfunding сравнивается с этой полной суммой.
- Break-even reference всегда запускается с `no_effect`; candidate и reference
  получают одинаковые stable random streams. Решение должно одновременно
  пройти paired-metric и liquidity/bankruptcy risk gates.
- Значение founder share не входит ни в одну целевую метрику.

## D-046. CLI, reporting flags и manifest

- `reporting.write_*` разрешает соответствующий формат, но не задаёт путь.
  Файл создаётся только при явном CLI path; попытка записать отключённый формат
  возвращает ошибку. Предупреждения всегда идут в stderr и не загрязняют JSON.
- Per-run terminal summaries включаются автоматически в deterministic mode и
  опционально через `-include-run-summaries` в Monte Carlo.
- `MANIFEST.sha256` не содержит хеш самого себя. Самореферентный SHA-256
  неустойчив по определению; это исключение явно записано в `doc/README.md`.

## D-047. Целочисленный найм, просрочка и shock follow-up

- В целочисленных headcount modes `gross_hiring_need`, `hire_capacity` и
  `affordable_hires` являются жёсткими верхними границами. Округление не может
  создать целого нанятого сотрудника, если доступного лимита меньше единицы.
- Неоплаченные обязательные расходы без собственной очереди (salary, payroll
  tax, fixed/variable/workforce/governance/shock costs и interest) переходят в
  `general mandatory arrears` на следующий месяц и оплачиваются раньше новых
  расходов. Непогашенный principal не дублируется в arrears: он остаётся в
  `debt_balance` и снова попадает в debt-service schedule.
- Shock survival оценивается только при наличии полного окна от месяца шока до
  `shock_month + 12`. Для промежуточного горизонта разрешено использовать
  последующие месяцы уже рассчитанного полного simulation path. Шоки без
  полного follow-up исключаются из знаменателя; если оцениваемых shock runs
  нет, JSON `shock_survival_rate` равен `null`, а CSV-ячейка остаётся пустой.
