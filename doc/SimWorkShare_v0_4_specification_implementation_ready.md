
---
title: "SimWorkShare v0.4 - implementation-ready specification"
subtitle: "Сравнение traditional_company, profit_sharing, employee_ownership_partial и worker_cooperative"
date: "2026-07-12"
lang: ru-RU
---

# SimWorkShare v0.4 - implementation-ready specification

**Версия документа:** 0.4  
**Статус:** implementation-ready specification  
**Назначение:** экономическая имитационная модель для сравнения организационных систем компании.  
**Принцип нейтральности:** модель не предполагает заранее превосходство кооператива, profit sharing или традиционной компании.

## 0. Анализ v0.3 и причины переработки

v0.3 была сфокусирована на сравнении fixed salary и profit sharing. Ее сильные элементы сохраняются: direct `behavior_case`, отделение P&L от cash-flow, AR/tax/bonus queues, restricted cash at accrual, common random numbers, paired deltas и sensitivity tests. Однако v0.3 не отвечает новой исследовательской цели, потому что не моделирует владение, участие работников в управлении, рост производственных возможностей, динамический headcount, ограничения внешнего капитала, концентрацию риска сотрудников и governance costs.

| Проблема v0.3 | Почему мешает v0.4 | Решение v0.4 |
|---|---|---|
| Объект сравнения - compensation scenarios | Новая цель сравнивает организационные системы | Ввести `organizational_scenarios` |
| `employees_count` постоянен | Нужны найм, увольнения, текучесть и итоговый размер | Dynamic workforce |
| Нет производственной мощности | Нельзя измерить долгосрочное развитие | `productive_capacity_revenue_monthly` и reinvestment lag |
| Profit sharing смешивает распределение и мотивацию | Ownership, profit distribution и governance должны быть отдельными механизмами | Независимые поля сценария |
| Нет governance costs/delay/quality | Участие в управлении может помогать или вредить | `GovernanceModel` |
| Главная метрика - owner cash | Новая цель исключает интересы основателя как критерий | Company/workforce/development/resilience metrics |
| Нет employee risk concentration | Доход и накопления могут зависеть от одной компании | `employee_risk_concentration_index` |
| Нет ограничений external capital | Они могут менять рост и устойчивость | `external_capital_access_multiplier` |

## 1. Цель и границы модели

Цель SimWorkShare v0.4 - проверить гипотезу: при каких условиях коллективное владение компанией, участие сотрудников в управлении и распределение результатов повышают производительность, устойчивость и темпы развития компании по сравнению с традиционной организацией.

Модель рассчитывает численность сотрудников, найм, увольнения, производительность, выручку, расходы, прибыль, cash-flow, распределение прибыли, реинвестирование, денежный резерв, growth capacity, motivation assumptions, ownership retention effects, governance costs, decision quality/delay, free rider, fairness, employee risk concentration, capital constraints и реакцию на market shocks.

Модель не является юридической моделью кооператива, налоговой консультацией, бухгалтерским балансом полного стандарта, оценкой стоимости доли основателя или доказательством причинности.

## 2. Исследовательские вопросы и гипотезы

| ID | Вопрос или гипотеза | Проверка |
|---|---|---|
| RQ1 | При каком productivity uplift worker cooperative не уступает traditional company? | Break-even по `sustainable_development_value_proxy` и risk constraints |
| RQ2 | Когда profit sharing дает эффект без ownership/governance? | Сравнить `profit_sharing` с `traditional_company` при behavior cases |
| RQ3 | Компенсирует ли снижение текучести governance costs? | Paired deltas turnover, hiring costs, cash, capacity |
| RQ4 | Когда fairness/free rider ухудшает outcome? | `negative_fairness_free_rider` sensitivity |
| RQ5 | Как governance влияет на shock response и investment efficiency? | Decision quality/delay formulas |
| RQ6 | Как risk concentration влияет на retention? | `risk_concentration_turnover_sensitivity_annual_pp` |
| RQ7 | Какие системы устойчивее при шоках? | Shock survival rate and post-shock recovery |
| H1 | Ownership может повысить retention | Только если задан отрицательный turnover delta |
| H2 | Ownership может повысить productivity | Только если задан positive sensitivity |
| H3 | Governance может повысить decision quality | Через explicit quality parameters |
| H4 | Governance может замедлять решения | Через `decision_delay_months` |
| H5 | Free rider может снизить productivity | Через `free_rider_penalty` |
| H6 | Reinvestment повышает capacity | Через investment cash outflow and lag |

## 3. Термины и определения

| Термин | Определение |
|---|---|
| `organizational_scenario` | Полный набор параметров устройства компании: владение, управление, распределение результата, реинвестирование, доступ к капиталу |
| `traditional_company` | Фиксированная зарплата, centralized management, сотрудники не участвуют в собственности и profit distribution |
| `profit_sharing` | Фиксированная зарплата плюс cash distribution из результата; ownership и governance остаются традиционными |
| `employee_ownership_partial` | Промежуточная система частичного ownership и ограниченного governance voice |
| `worker_cooperative` | Коллективное владение сотрудниками, participatory governance, распределение части результата и обязательное reinvestment/reserve policy |
| `productive_capacity_revenue_monthly` | Максимальная месячная выручка, которую компания может произвести при достаточном спросе и труде |
| `labor_revenue_capacity` | Потенциальная выручка, ограниченная численностью и productivity |
| `reinvestment` | Денежный outflow на развитие, создающий capacity после лага |
| `member_capital_account` | Внутренний счет сотрудника, связанный с компанией и создающий employee risk concentration |
| `common random numbers` | Одинаковая внешняя траектория для сравниваемых сценариев в одном Monte Carlo run |

## 4. Сценарии организационного устройства

| Сценарий | Зарплата | Владение | Распределение результата | Управление | Реинвестирование | Доступ к капиталу |
|---|---|---|---|---|---|---|
| `traditional_company` | fixed | 0 | 0 | centralized | retained earnings policy | full |
| `profit_sharing` | fixed | 0 | cash distribution | centralized | company policy | full |
| `employee_ownership_partial` | fixed | partial | cash + optional member capital | partial voice | explicit | moderately constrained |
| `worker_cooperative` | fixed or stable base wage | collective | cash + member capital | participatory | explicit and protected | constrained |


Механизмы не смешиваются. Формально `employee_ownership_fraction`, `employee_cash_distribution_rate`, `member_capital_allocation_rate`, `governance_participation_intensity` и `reinvestment_rate` являются независимыми параметрами.

## 5. Полный перечень входных параметров

Все `rate` задаются десятичной дробью: `0.10 = 10 percent`. Параметры с суффиксом `_pp` - absolute percentage points годовой ставки. Денежные величины выражаются в `simulation.currency`.

### 5.1. simulation
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `schema_version` | string | version | 0.4 | `0.4` | Версия схемы конфигурации. |
| `mode` | enum | mode | deterministic|monte_carlo | `monte_carlo` | Режим расчета. |
| `months` | integer | months | 1 to 240 | `240` | Горизонт моделирования. |
| `horizons_months` | array[int] | months | 1 to months | `[60,120,240]` | Горизонты отчета 5/10/20 лет. |
| `runs` | integer | runs | 1 to 1000000 | `1000` | Количество Monte Carlo прогонов. |
| `random_seed` | integer | seed | 0 to 2^63-1 | `42` | Воспроизводимость. |
| `common_random_numbers` | bool | bool | true|false | `True` | Одинаковая внешняя траектория для сценариев. |
| `currency` | string | currency | text | `RUB` | Валюта денежных параметров. |
| `epsilon` | number | currency/share | 0 to 1e-3 | `1e-9` | Числовая погрешность. |
| `headcount_mode` | enum | mode | fractional|integer_expected|integer_random | `fractional` | Режим дискретности headcount. |
| `stop_after_bankruptcy` | bool | bool | true|false | `True` | Банкротство как поглощающее событие. |

### 5.2. company_economics
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `initial_headcount` | number/integer | people | >0 | `50` | Начальная численность. |
| `base_salary_per_employee_monthly` | number/integer | currency/month/person | >=0 | `100000` | Фиксированная зарплата. |
| `salary_payroll_tax_rate` | number/integer | share | 0 to 1 | `0.0` | Взносы/налоги работодателя на зарплату. |
| `standard_hours_per_employee_month` | number/integer | hours/person/month | 1 to 300 | `160` | Норма часов. |
| `base_revenue_per_effective_employee_monthly` | number/integer | currency/month/effective person | >=0 | `230000` | Базовая выработка full-productivity employee. |
| `initial_market_demand_monthly` | number/integer | currency/month | >=0 | `11500000` | Начальный внешний спрос. |
| `initial_productive_capacity_revenue_monthly` | number/integer | currency/month | >=0 | `13800000` | Начальная производственная мощность. |
| `fixed_costs_monthly` | number/integer | currency/month | >=0 | `2000000` | Постоянные расходы кроме payroll. |
| `variable_cost_rate` | number/integer | share of revenue | 0 to 1 | `0.25` | Переменные расходы. |
| `profit_tax_rate` | number/integer | share | 0 to 1 | `0.2` | Упрощенный налог на прибыль. |
| `profit_tax_payment_lag_months` | number/integer | months | 0 to 24 | `1` | Лаг оплаты налога. |
| `cost_inflation_monthly` | number/integer | monthly rate | -0.05 to 0.10 | `0.005` | Инфляция зарплат и fixed costs. |
| `capacity_depreciation_rate_monthly` | number/integer | share/month | 0 to 0.10 | `0.002` | Устаревание производственных возможностей. |
| `capacity_revenue_created_per_currency_invested` | number/integer | currency monthly capacity / currency invested | >=0 | `0.08` | Эффективность инвестиций в capacity. |
| `investment_activation_lag_months` | number/integer | months | 0 to 60 | `3` | Лаг ввода capacity. |
| `required_cash_reserve_months` | number/integer | months | 0 to 24 | `2.0` | Резерв обязательных платежей. |
| `starting_cash` | number/integer | currency | >=0 | `15000000` | Денежный запас на старт. |
| `opening_accounts_receivable` | number/integer | currency | >=0 | `1725000` | Дебиторка на старт. |

### 5.3. market
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `market_process` | number/enum/array | process | deterministic|bounded_lognormal | `bounded_lognormal` | Процесс спроса. |
| `market_growth_monthly` | number/enum/array | monthly rate | -0.10 to 0.10 | `0.003` | Средний рост спроса. |
| `market_volatility_monthly` | number/enum/array | std dev | 0 to 1 | `0.08` | Волатильность спроса. |
| `market_factor_min` | number/enum/array | multiplier | 0 to 10 | `0.5` | Нижняя граница market factor. |
| `market_factor_max` | number/enum/array | multiplier | 0 to 10 | `1.8` | Верхняя граница market factor. |
| `seasonality_multipliers` | number/enum/array | multiplier | 12 values, 0 to 10 | `[1 x 12]` | Сезонность спроса. |
| `revenue_collection_rate_current_month` | number/enum/array | share | 0 to 1 | `0.85` | Доля выручки, собранная cash в месяце. |
| `accounts_receivable_lag_months` | number/enum/array | months | 0 to 24 | `1` | Лаг сбора AR. |
| `bad_debt_rate` | number/enum/array | share | 0 to 1 | `0.0` | Безнадежная дебиторка. |
| `shock_probability_monthly` | number/enum/array | probability | 0 to 1 | `0.03` | Вероятность рыночного шока. |
| `shock_revenue_multiplier` | number/enum/array | multiplier | 0 to 1 | `0.8` | Базовый множитель выручки при шоке. |
| `shock_cost_mean` | number/enum/array | currency | >=0 | `0` | Средний cost шока. |
| `shock_cost_std` | number/enum/array | currency | >=0 | `0` | Стандартное отклонение shock cost. |
| `cash_collection_stress_multiplier` | number/enum/array | multiplier | 0 to 1 | `0.9` | Снижение collection rate при шоке. |
| `labor_market_factor` | number/enum/array | multiplier | 0 to 5 | `1.0` | Множитель текучести. |
| `credit_market_factor` | number/enum/array | multiplier | 0 to 5 | `1.0` | Доступность внешнего капитала. |

### 5.4. workforce
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `base_turnover_rate_annual` | number/enum/array | annual share | 0 to 1 | `0.2` | Базовая годовая текучесть. |
| `min_turnover_rate_annual` | number/enum/array | annual share | 0 to 1 | `0.03` | Нижняя граница. |
| `max_turnover_rate_annual` | number/enum/array | annual share | 0 to 1 | `0.6` | Верхняя граница. |
| `turnover_random_mode` | number/enum/array | mode | deterministic|binomial | `deterministic` | Стохастика увольнений. |
| `high_performer_share` | number/enum/array | share | 0 to 1 | `0.2` | Доля сильных сотрудников для risk indicator. |
| `recruiting_cost_per_hire` | number/enum/array | currency/hire | >=0 | `50000` | Стоимость найма. |
| `onboarding_cost_per_hire` | number/enum/array | currency/hire | >=0 | `50000` | Адаптация/обучение. |
| `manager_time_cost_per_hire` | number/enum/array | currency/hire | >=0 | `25000` | Время менеджмента. |
| `exit_admin_cost_per_leaver` | number/enum/array | currency/leaver | >=0 | `10000` | Админ. стоимость ухода. |
| `lost_productivity_cost_per_leaver` | number/enum/array | currency/leaver | >=0 | `0` | Явная стоимость потерь; default 0 чтобы не double count. |
| `severance_cost_per_layoff` | number/enum/array | currency/layoff | >=0 | `100000` | Стоимость сокращения. |
| `ramp_duration_months` | number/enum/array | months | 0 to 24 | `3` | Период выхода на продуктивность. |
| `ramp_productivity_multipliers` | number/enum/array | multiplier | 0 to 1 | `[0.5,0.75,0.9]` | Продуктивность новых сотрудников. |
| `max_hires_per_month_rate` | number/enum/array | share of headcount | 0 to 1 | `0.1` | Ограничение найма. |
| `max_layoffs_per_month_rate` | number/enum/array | share of headcount | 0 to 1 | `0.1` | Ограничение сокращений. |
| `layoff_trigger_cash_ratio` | number/enum/array | ratio | 0 to 10 | `0.5` | Порог cash stress. |
| `leaver_paid_fraction_of_month` | number/enum/array | share | 0 to 1 | `0.5` | Доля месяца, оплаченная уходящим. |
| `new_hire_paid_fraction_of_month` | number/enum/array | share | 0 to 1 | `0.5` | Доля месяца, оплаченная новым. |
| `min_productivity_uplift` | number/enum/array | uplift | -1 to 1 | `-0.15` | Нижняя граница uplift. |
| `max_productivity_uplift` | number/enum/array | uplift | -1 to 1 | `0.2` | Верхняя граница uplift. |
| `turnover_productivity_penalty_per_annual_turnover` | number/enum/array | uplift loss | 0 to 10 | `0.1` | Потеря productivity от excess turnover. |
| `target_staffing_buffer` | number/enum/array | multiplier | 0 to 3 | `1.05` | Запас headcount под спрос. |
| `max_cash_share_for_hiring` | number/enum/array | share | 0 to 1 | `0.25` | Доля excess cash на hiring. |

### 5.5. employee_risk
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `employee_external_savings_proxy_per_employee` | number | currency/person | >=0 | `600000` | Proxy внешних накоплений сотрудника. |
| `employment_dependence_index` | number | index | 0 to 1 | `1.0` | Зависимость занятости от компании. |
| `risk_weight_variable_income` | number | weight | 0 to 1 | `0.4` | Вес переменного дохода. |
| `risk_weight_member_capital` | number | weight | 0 to 1 | `0.4` | Вес member capital. |
| `risk_weight_employment_dependence` | number | weight | 0 to 1 | `0.2` | Вес зависимости занятости. |

### 5.6. financing
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `base_credit_line` | number/enum | currency | >=0 | `0` | Доступная кредитная линия. |
| `debt_interest_rate_annual` | number/enum | annual rate | 0 to 1 | `0.18` | Стоимость долга. |
| `scheduled_principal_payment_monthly` | number/enum | currency/month | >=0 | `0` | Плановое погашение principal. |
| `external_growth_capital_limit_monthly` | number/enum | currency/month | >=0 | `0` | Максимальный внешний капитал на развитие. |
| `external_capital_type` | number/enum | type | debt|non_dilutive_grant | `debt` | Тип внешнего growth capital. |
| `distribution_payroll_tax_rate` | number/enum | share | 0 to 1 | `0.0` | Налоги/взносы на employee distribution. |
| `distribution_tax_deductible_share` | number/enum | share | 0 to 1 | `1.0` | Доля distribution, уменьшающая taxable profit. |
| `employee_distribution_payout_lag_months` | number/enum | months | 0 to 24 | `1` | Лаг выплаты employee distribution. |
| `member_capital_redemption_lag_months` | number/enum | months | 0 to 120 | `24` | Лаг выплаты member capital при уходе. |
| `member_capital_redemption_fraction_on_exit` | number/enum | share | 0 to 1 | `1.0` | Доля member capital к redemption. |
| `reserve_release_rate_on_stress` | number/enum | share/month | 0 to 1 | `0.5` | Release restricted reserve under stress. |

### 5.7. organizational_scenarios
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `name` | number/string/enum/array/null | id | unique snake_case | `required` | Имя сценария. |
| `system_type` | number/string/enum/array/null | type | traditional_company|profit_sharing|employee_ownership_partial|worker_cooperative | `required` | Тип системы. |
| `employee_ownership_fraction` | number/string/enum/array/null | share | 0 to 1 | `scenario-specific` | Экономическое участие сотрудников. |
| `employee_cash_distribution_rate` | number/string/enum/array/null | share | 0 to 1 | `scenario-specific` | Cash distribution сотрудникам. |
| `member_capital_allocation_rate` | number/string/enum/array/null | share | 0 to 1 | `scenario-specific` | Начисление на внутренние счета. |
| `reinvestment_rate` | number/string/enum/array/null | share | 0 to 1 | `scenario-specific` | Доля результата на развитие. |
| `organizational_reserve_rate` | number/string/enum/array/null | share | 0 to 1 | `scenario-specific` | Резерв устойчивости. |
| `external_distribution_rate` | number/string/enum/array/null | share | 0 to 1 | `0` | Cash outflow вне компании; не target metric. |
| `result_hurdle_monthly` | number/string/enum/array/null | currency/month | >=0 | `0` | Результат, не подлежащий распределению. |
| `allocation_priority` | number/string/enum/array/null | order | valid allocation ids | `required` | Очередь распределения результата. |
| `distribution_rule` | number/string/enum/array/null | rule | none|equal_per_capita|contribution_weighted|hybrid | `scenario-specific` | Правило распределения. |
| `contribution_measurement_quality` | number/string/enum/array/null | index | 0 to 1 | `scenario-specific` | Качество измерения индивидуального вклада. |
| `peer_monitoring_effectiveness` | number/string/enum/array/null | index | 0 to 1 | `scenario-specific` | Снижение free-rider через peer monitoring. |
| `transparency_index` | number/string/enum/array/null | index | 0 to 1 | `scenario-specific` | Прозрачность формулы и финансов. |
| `employment_stabilization_preference` | number/string/enum/array/null | index | 0 to 1 | `scenario-specific` | Склонность избегать layoffs. |
| `external_capital_access_multiplier` | number/string/enum/array/null | multiplier | 0 to 5 | `scenario-specific` | Ограничение/преимущество внешнего капитала. |
| `profit_distribution_period_months` | number/string/enum/array/null | months | 1 to 12 | `1` | Период начисления distribution. |
| `max_distribution_per_employee_period` | number/string/enum/array/null | currency/person/period | >=0|null | `None` | Cap выплаты. |
| `behavior_case_refs` | number/string/enum/array/null | ids | non-empty | `required` | Какие behavior cases прогонять. |

### 5.8. governance
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `governance_participation_intensity` | number | varies | see spec | `0.8` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `base_governance_hours_per_employee_month` | number | varies | see spec | `4.0` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `fixed_governance_hours_monthly` | number | varies | see spec | `40.0` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `governance_cash_cost_fixed_monthly` | number | varies | see spec | `100000` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `governance_cash_cost_per_employee_monthly` | number | varies | see spec | `2000` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `decision_complexity_index` | number | varies | see spec | `1.2` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `base_decision_delay_months` | number | varies | see spec | `0.25` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `delay_per_participation_months` | number | varies | see spec | `0.5` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `local_autonomy_index` | number | varies | see spec | `0.6` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `decentralization_speed_gain_months` | number | varies | see spec | `0.2` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `governance_capability_index` | number | varies | see spec | `1.0` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `information_sharing_quality` | number | varies | see spec | `0.8` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `trust_index` | number | varies | see spec | `0.7` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `quality_gain_from_participation` | number | varies | see spec | `0.03` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `coordination_loss_from_participation` | number | varies | see spec | `0.02` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `conflict_loss_sensitivity` | number | varies | see spec | `0.02` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `decision_delay_quality_loss` | number | varies | see spec | `0.01` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `decision_quality_min` | number | varies | see spec | `0.7` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `decision_quality_max` | number | varies | see spec | `1.2` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `shock_mitigation_sensitivity` | number | varies | see spec | `0.15` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `shock_delay_amplification` | number | varies | see spec | `0.03` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |
| `investment_efficiency_sensitivity` | number | varies | see spec | `0.15` | Параметр модели управления; влияет на hours, cost, delay, quality, shock response или investment efficiency. |

### 5.9. behavior_cases
| Имя | Тип | Ед. | Диапазон | Default | Экономический смысл |
|---|---|---|---|---|---|
| `base_productivity_uplift_direct` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `ownership_productivity_sensitivity` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `profit_distribution_productivity_sensitivity` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `governance_voice_productivity_sensitivity` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `base_turnover_delta_annual_pp` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `ownership_retention_delta_annual_pp_per_full_ownership` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `profit_distribution_retention_delta_annual_pp_per_10pp` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `governance_retention_delta_annual_pp_per_full_participation` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `fairness_base` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `transparency_to_fairness` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `equal_distribution_fairness_effect` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `contribution_based_distribution_fairness_effect` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `pay_dispersion_fairness_penalty` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `unpaid_governance_burden_penalty` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `zero_distribution_fairness_penalty` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `fairness_productivity_sensitivity` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `fairness_turnover_sensitivity_annual_pp` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `free_rider_base_penalty` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `free_rider_size_exponent` | number | varies | see spec | `0.5` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `free_rider_reference_headcount` | number | varies | see spec | `50` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `free_rider_max_size_multiplier` | number | varies | see spec | `3.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `risk_concentration_turnover_sensitivity_annual_pp` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `income_volatility_turnover_sensitivity_annual_pp` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |
| `high_performer_attrition_delta_pp` | number | varies | see spec | `0.0` | Сценарное допущение поведения; в no_effect равно нейтральному значению. |



## 6. Состояние модели на начало месяца

| Группа | State variables на начало месяца `t` |
|---|---|
| `EnvironmentState` | `month`, `market_trend_t`, `market_factor_t`, `seasonality_multiplier_t`, `shock_happened_t`, `shock_cost_t`, `collection_rate_multiplier_t`, `labor_market_factor_t`, `credit_market_factor_t` |
| `WorkforceState` | `headcount_begin_t`, `ramp_cohorts_begin_t[k]`, `full_productivity_headcount_begin_t`, `income_history_12m`, `zero_distribution_streak`, `last_turnover_rate_annual` |
| `GovernanceState` | `governance_capability_index_t`, `trust_index_t`, `information_sharing_quality_t`, `decision_delay_months_t`, `decision_quality_multiplier_t` |
| `ProductionState` | `productive_capacity_revenue_monthly_begin_t`, `capacity_addition_queue`, `capacity_depreciation_rate_monthly` |
| `FinancialState` | `cash_total_begin_t`, `restricted_distribution_cash_begin_t`, `restricted_reserve_cash_begin_t`, `debt_balance_begin_t`, `ar_queue`, `tax_payable_queue`, `employee_distribution_payable_queue`, `member_capital_redemption_queue` |
| `MemberCapitalState` | `member_capital_accounts_total_begin_t`, `member_capital_per_employee_average_begin_t` |
| `RiskState` | `min_unrestricted_cash_to_date`, `bankruptcy_absorbed_flag`, `months_since_first_shock`, `unpaid_mandatory_obligations_to_date` |

Definitions:

```text
restricted_cash_total_t = restricted_distribution_cash_t + restricted_reserve_cash_t
unrestricted_cash_t = cash_total_t - restricted_cash_total_t
```

## 7. Точный порядок расчетов внутри месяца

1. Если `bankruptcy_absorbed_flag=true` и `stop_after_bankruptcy=true`, записать inactive month.
2. Получить `EnvironmentMonth[t]` из common environment path.
3. Ввести due capacity additions.
4. Рассчитать governance hours, costs, decision delay и decision quality.
5. Рассчитать fairness, free rider, employee risk concentration и behavioral effects.
6. Рассчитать voluntary turnover.
7. Рассчитать layoffs на основе beginning cash stress и employment stabilization policy.
8. Рассчитать desired headcount, hiring need, hiring affordability и hires.
9. Обновить ramp cohorts для production текущего месяца.
10. Рассчитать effective employees и productivity multiplier.
11. Рассчитать market demand, labor revenue capacity, productive capacity limit и revenue.
12. Рассчитать accrual operating costs.
13. Начислить AR и собрать cash из текущей выручки и прошлой AR.
14. Рассчитать due obligations: taxes, employee distributions, member redemptions, debt service.
15. Оплатить обязательные cash obligations.
16. При необходимости высвободить restricted reserve по stress policy.
17. При необходимости привлечь credit line для mandatory cash gap.
18. Рассчитать operating profit before allocation.
19. Рассчитать cash-safe base for allocations.
20. По `allocation_priority` распределить positive result.
21. Рассчитать taxable profit, profit tax accrual и поставить tax в queue.
22. Поставить employee distribution в payable queue и зарезервировать cash.
23. Провести cash reinvestment и optional external growth capital.
24. Поставить capacity additions в queue с лагом.
25. Рассчитать closing cash, restricted cash, debt, member capital, headcount.
26. Рассчитать risk flags.
27. Сохранить `MonthlyResult`.

## 8. Формулы для каждого этапа

### 8.1. Environment path

```text
market_trend_t = market_trend_{t-1} * (1 + market_growth_monthly)
raw_market_factor_t = exp((-0.5 * sigma_m^2) + sigma_m * z_market_t)
market_factor_t = clamp(raw_market_factor_t, market_factor_min, market_factor_max)
shock_happened_t ~ Bernoulli(shock_probability_monthly)
collection_rate_multiplier_t = cash_collection_stress_multiplier if shock_happened_t else 1
```

In deterministic mode: `market_factor_t=1`, no random shocks except explicit schedule, `runs=1`.

### 8.2. Governance cost and decision quality

```text
governance_hours_t = headcount_begin_t * governance_participation_intensity * base_governance_hours_per_employee_month + fixed_governance_hours_monthly
governance_admin_equivalent_employees_t = governance_hours_t / standard_hours_per_employee_month
governance_cash_cost_t = governance_cash_cost_fixed_monthly + headcount_begin_t * governance_cash_cost_per_employee_monthly

decision_delay_months_t = max(0,
  base_decision_delay_months
  + decision_complexity_index * governance_participation_intensity * delay_per_participation_months / governance_capability_index
  - local_autonomy_index * decentralization_speed_gain_months
)

decision_quality_raw_t = 1
  + quality_gain_from_participation * governance_participation_intensity * information_sharing_quality
  - coordination_loss_from_participation * governance_participation_intensity * decision_complexity_index
  - conflict_loss_sensitivity * (1 - trust_index)
  - decision_delay_quality_loss * decision_delay_months_t

decision_quality_multiplier_t = clamp(decision_quality_raw_t, decision_quality_min, decision_quality_max)
```

### 8.3. Fairness

```text
distribution_rule_effect_t =
  0 for none
  equal_distribution_fairness_effect for equal_per_capita
  contribution_based_distribution_fairness_effect * contribution_measurement_quality for contribution_weighted
  0.5 * equal_distribution_fairness_effect + 0.5 * contribution_based_distribution_fairness_effect * contribution_measurement_quality for hybrid

governance_burden_share_t = governance_hours_t / max(epsilon, headcount_begin_t * standard_hours_per_employee_month)

fairness_index_t = clamp(
  fairness_base + transparency_to_fairness * transparency_index + distribution_rule_effect_t
  - pay_dispersion_fairness_penalty * pay_dispersion_index_t
  - unpaid_governance_burden_penalty * governance_burden_share_t
  - zero_distribution_fairness_penalty * zero_distribution_streak_t,
  -1, 1
)
```

Default `pay_dispersion_index_t=0` unless explicitly configured.

### 8.4. Free rider

```text
collective_distribution_exposure_t = employee_cash_distribution_rate + member_capital_allocation_rate
equal_distribution_factor_t = 1 for equal_per_capita, 0.5 for hybrid, 0 otherwise
size_factor_t = min(free_rider_max_size_multiplier, (headcount_begin_t / free_rider_reference_headcount) ^ free_rider_size_exponent)
monitoring_reduction_t = peer_monitoring_effectiveness * contribution_measurement_quality
free_rider_penalty_t = free_rider_base_penalty * collective_distribution_exposure_t * equal_distribution_factor_t * size_factor_t * (1 - monitoring_reduction_t)
```

### 8.5. Employee risk concentration

```text
variable_income_share_12m_t = employee_cash_distribution_paid_12m_t / max(epsilon, fixed_salary_paid_12m_t + employee_cash_distribution_paid_12m_t)
member_capital_per_employee_t = member_capital_accounts_total_begin_t / max(epsilon, headcount_begin_t)
capital_concentration_t = member_capital_per_employee_t / max(epsilon, member_capital_per_employee_t + employee_external_savings_proxy_per_employee)
employee_risk_concentration_index_t = clamp(
  risk_weight_variable_income * variable_income_share_12m_t
  + risk_weight_member_capital * capital_concentration_t
  + risk_weight_employment_dependence * employment_dependence_index,
  0, 1
)
```

### 8.6. Productivity uplift

```text
ownership_salience_t = employee_ownership_fraction * transparency_index
profit_distribution_salience_t = employee_cash_distribution_rate / 0.10
governance_salience_t = governance_participation_intensity

motivation_uplift_raw_t = base_productivity_uplift_direct
  + ownership_productivity_sensitivity * ownership_salience_t
  + profit_distribution_productivity_sensitivity * profit_distribution_salience_t
  + governance_voice_productivity_sensitivity * governance_salience_t

fairness_productivity_effect_t = fairness_productivity_sensitivity * fairness_index_t
turnover_excess_t = max(0, turnover_rate_annual_t - base_turnover_rate_annual)
turnover_productivity_loss_t = turnover_productivity_penalty_per_annual_turnover * turnover_excess_t

productivity_uplift_t = clamp(
  motivation_uplift_raw_t + fairness_productivity_effect_t
  + governance_voice_productivity_sensitivity * max(0, decision_quality_multiplier_t - 1)
  - free_rider_penalty_t - turnover_productivity_loss_t,
  min_productivity_uplift, max_productivity_uplift
)
productivity_multiplier_t = 1 + productivity_uplift_t
```

### 8.7. Turnover

```text
fairness_turnover_delta_t = - fairness_turnover_sensitivity_annual_pp * fairness_index_t
risk_turnover_delta_t = risk_concentration_turnover_sensitivity_annual_pp * employee_risk_concentration_index_t
income_volatility_turnover_delta_t = income_volatility_turnover_sensitivity_annual_pp * income_volatility_index_12m_t
ownership_turnover_delta_t = ownership_retention_delta_annual_pp_per_full_ownership * ownership_salience_t
distribution_turnover_delta_t = profit_distribution_retention_delta_annual_pp_per_10pp * profit_distribution_salience_t
governance_turnover_delta_t = governance_retention_delta_annual_pp_per_full_participation * governance_salience_t

turnover_rate_annual_t = clamp(
  base_turnover_rate_annual * labor_market_factor_t
  + base_turnover_delta_annual_pp
  + ownership_turnover_delta_t + distribution_turnover_delta_t + governance_turnover_delta_t
  + fairness_turnover_delta_t + risk_turnover_delta_t + income_volatility_turnover_delta_t,
  min_turnover_rate_annual, max_turnover_rate_annual
)
turnover_rate_monthly_t = 1 - (1 - turnover_rate_annual_t)^(1/12)
voluntary_leavers_t = headcount_begin_t * turnover_rate_monthly_t  # deterministic mode
```

### 8.8. Layoffs and hiring

```text
required_cash_reserve_begin_t = required_cash_reserve_months * (salary_costs_reference_monthly_t + fixed_costs_reference_monthly_t + scheduled_debt_service_reference_monthly_t)
begin_cash_stress_ratio_t = unrestricted_cash_begin_t / max(epsilon, required_cash_reserve_begin_t)
cash_stress_layoff_pressure_t = max(0, layoff_trigger_cash_ratio - begin_cash_stress_ratio_t) / max(epsilon, layoff_trigger_cash_ratio)
distress_layoff_rate_t = max_layoffs_per_month_rate * cash_stress_layoff_pressure_t * (1 - employment_stabilization_preference)
distress_layoffs_t = min(headcount_begin_t - voluntary_leavers_t, (headcount_begin_t - voluntary_leavers_t) * distress_layoff_rate_t)

target_revenue_for_staffing_t = min(market_demand_forecast_t * target_staffing_buffer, productive_capacity_revenue_monthly_begin_t)
desired_headcount_t = target_revenue_for_staffing_t / (base_revenue_per_effective_employee_monthly * max(0.10, productivity_multiplier_t))
gross_hiring_need_t = max(0, desired_headcount_t - headcount_after_exits_t)
hire_capacity_t = max_hires_per_month_rate * max(1, headcount_begin_t)
hiring_cash_available_t = max(0, unrestricted_cash_begin_t - required_cash_reserve_begin_t) * max_cash_share_for_hiring
affordable_hires_t = hiring_cash_available_t / max(epsilon, recruiting_cost_per_hire + onboarding_cost_per_hire + manager_time_cost_per_hire)
hires_t = min(gross_hiring_need_t, hire_capacity_t, affordable_hires_t)
headcount_end_t = headcount_begin_t - voluntary_leavers_t - distress_layoffs_t + hires_t
```

### 8.9. Effective employees and revenue

```text
effective_ramping_employees_t = sum_k ramp_cohort_after_exits_t[k] * ramp_productivity_multipliers[k]
effective_new_hires_t = hires_t * ramp_productivity_multipliers[0]
effective_employees_t = max(0, full_productivity_headcount_after_exits_t + effective_ramping_employees_t + effective_new_hires_t - governance_admin_equivalent_employees_t)

base_shock_loss_t = 1 - shock_revenue_multiplier
shock_loss_multiplier_t = clamp(1 - shock_mitigation_sensitivity * (decision_quality_multiplier_t - 1) + shock_delay_amplification * decision_delay_months_t, 0, 3)
effective_shock_revenue_multiplier_t = 1 - clamp(base_shock_loss_t * shock_loss_multiplier_t, 0, 1)

market_demand_t = initial_market_demand_monthly * market_trend_t * seasonality_multiplier_t * market_factor_t * effective_shock_revenue_multiplier_t
labor_revenue_capacity_t = effective_employees_t * base_revenue_per_effective_employee_monthly * productivity_multiplier_t
productive_capacity_limit_t = productive_capacity_revenue_monthly_begin_t
revenue_t = max(0, min(market_demand_t, labor_revenue_capacity_t, productive_capacity_limit_t))
```

### 8.10. Costs, cash collection and mandatory payments

```text
paid_employees_t = headcount_begin_t - leaver_paid_fraction_of_month * voluntary_leavers_t - leaver_paid_fraction_of_month * distress_layoffs_t + new_hire_paid_fraction_of_month * hires_t
salary_cost_t = paid_employees_t * base_salary_per_employee_monthly * cumulative_cost_inflation_t
salary_payroll_tax_t = salary_cost_t * salary_payroll_tax_rate
fixed_costs_t = fixed_costs_monthly * cumulative_cost_inflation_t
variable_costs_t = revenue_t * variable_cost_rate
hiring_cost_t = hires_t * (recruiting_cost_per_hire + onboarding_cost_per_hire + manager_time_cost_per_hire)
exit_cost_t = voluntary_leavers_t * (exit_admin_cost_per_leaver + lost_productivity_cost_per_leaver)
layoff_cost_t = distress_layoffs_t * severance_cost_per_layoff
operating_costs_before_allocation_t = salary_cost_t + salary_payroll_tax_t + fixed_costs_t + variable_costs_t + hiring_cost_t + exit_cost_t + layoff_cost_t + governance_cash_cost_t + shock_cost_accrual_t

cash_collected_current_t = revenue_t * clamp(revenue_collection_rate_current_month * collection_rate_multiplier_t, 0, 1)
new_ar_t = revenue_t * (1 - effective_collection_rate_t) * (1 - bad_debt_rate)
cash_collected_from_revenue_t = cash_collected_current_t + ar_queue.amount_due(t)
```

```text
mandatory_cash_payments_t = salary_cost_t + salary_payroll_tax_t + fixed_costs_t + variable_costs_t + turnover_and_workforce_cost_t + governance_cash_cost_t + shock_cost_accrual_t + taxes_due_t + employee_distribution_due_t + member_capital_redemption_due_t + interest_expense_t + principal_payment_t
cash_after_mandatory_before_financing_t = cash_total_begin_t + cash_collected_from_revenue_t - mandatory_cash_payments_t
credit_draw_for_liquidity_t = min(max(0, -cash_after_mandatory_before_financing_t), credit_headroom_t)
cash_after_mandatory_t = cash_after_mandatory_before_financing_t + credit_draw_for_liquidity_t
```

### 8.11. Result allocation, tax, reinvestment and capacity growth

```text
operating_profit_before_allocation_t = revenue_t - operating_costs_before_allocation_t
profit_before_tax_before_distribution_t = operating_profit_before_allocation_t - interest_expense_t
positive_result_base_t = max(0, profit_before_tax_before_distribution_t - result_hurdle_monthly)

cash_safe_allocation_budget_t = max(0, unrestricted_cash_before_allocations_t - required_cash_reserve_t - tax_reserve_estimate_t)

raw_employee_cash_distribution_gross_t = employee_cash_distribution_rate * positive_result_base_t
raw_member_capital_allocation_t = member_capital_allocation_rate * positive_result_base_t
raw_reinvestment_t = reinvestment_rate * positive_result_base_t
raw_organizational_reserve_t = organizational_reserve_rate * positive_result_base_t
raw_external_distribution_t = external_distribution_rate * positive_result_base_t
```

Validation: `employee_cash_distribution_rate + member_capital_allocation_rate + reinvestment_rate + organizational_reserve_rate + external_distribution_rate <= 1`.

For each item in `allocation_priority`:

```text
actual_i_t = min(raw_i_t, remaining_allocation_budget_t / cash_multiplier_i)
remaining_allocation_budget_t -= actual_i_t * cash_multiplier_i
```

Cash multipliers: employee cash distribution uses `1 + distribution_payroll_tax_rate`; other allocation types use `1`.

```text
deductible_employee_distribution_t = actual_employee_cash_distribution_gross_t * distribution_tax_deductible_share
taxable_profit_t = max(0, profit_before_tax_before_distribution_t - deductible_employee_distribution_t)
profit_tax_accrual_t = taxable_profit_t * profit_tax_rate

actual_reinvestment_total_t = cash_reinvestment_paid_t + external_growth_capital_draw_t
investment_efficiency_multiplier_t = clamp(1 + investment_efficiency_sensitivity * (decision_quality_multiplier_t - 1), 0, 3)
capacity_added_by_investment_t = actual_reinvestment_total_t * capacity_revenue_created_per_currency_invested * investment_efficiency_multiplier_t
effective_investment_lag_t = investment_activation_lag_months + round(decision_delay_months_t)
capacity_addition_queue.add(t + effective_investment_lag_t, capacity_added_by_investment_t)
```

### 8.12. Closing state and bankruptcy

```text
restricted_distribution_cash_close_t = restricted_distribution_cash_begin_t - employee_distribution_due_t + restricted_distribution_cash_new_t
restricted_reserve_cash_close_t = restricted_reserve_cash_after_release_t + actual_organizational_reserve_t
cash_total_close_t = cash_total_after_growth_capital_t
unrestricted_cash_close_t = cash_total_close_t - restricted_distribution_cash_close_t - restricted_reserve_cash_close_t

reserve_breach_flag_t = unrestricted_cash_close_t < required_cash_reserve_t
cash_gap_flag_t = cash_total_close_t < 0
liquidity_deficit_flag_t = unpaid_mandatory_obligations_t > epsilon
bankruptcy_flag_t = liquidity_deficit_flag_t OR debt_balance_close_t > credit_line_limit_t + epsilon
```

## 9. Правила распределения прибыли и реинвестирования

Распределение выполняется только из `positive_result_base_t` и только в пределах `cash_safe_allocation_budget_t`. Положительная accounting profit не означает автоматическую выплату.

| Сценарий | Default policy |
|---|---|
| `traditional_company` | `employee_cash_distribution_rate=0`, `member_capital_allocation_rate=0`, `governance=0`, `reinvestment_rate=0.15`, `organizational_reserve_rate=0.05` |
| `profit_sharing` | `employee_cash_distribution_rate=0.10`, ownership=0, governance=0, `reinvestment_rate=0.10`, reserve=0.05 |
| `employee_ownership_partial` | ownership=0.30, cash distribution=0.10, member capital=0.05, governance=0.30, reinvestment=0.15, reserve=0.05 |
| `worker_cooperative` | ownership=1.00, cash distribution=0.15, member capital=0.05, governance=0.80, reinvestment=0.25, reserve=0.10 |

## 10. Поведенческие механизмы

Поведение задается через explicit behavior cases. Модель не считает motivation effect установленным фактом.

| Case | Смысл |
|---|---|
| `no_effect` | Ownership, profit sharing и governance не дают productivity/retention эффекта |
| `retention_only` | Влияет только на удержание |
| `moderate_positive` | Умеренный положительный uplift/retention |
| `governance_costly` | Participation создает costs/delay без productivity gain |
| `negative_fairness_free_rider` | Fairness/free rider ухудшают productivity и turnover |
| `risk_concentration_negative` | Variable income и member capital повышают turnover |

| Положительный механизм | Ограничивающий механизм |
|---|---|
| Ownership может повысить commitment | Concentrated risk может повысить turnover |
| Profit distribution может повысить effort | Variable income volatility может снизить attractiveness |
| Governance voice может повысить качество решений | Governance может замедлить решения и снизить productive hours |
| Equal distribution может повысить cohesion | Equal distribution может демотивировать high performers |
| Member capital может удерживать сотрудников | Member capital создает redemption liabilities и employee risk |
| Reinvestment повышает capacity | Reinvestment снижает short-term cash |
| Employment stabilization снижает layoffs | Может увеличить bankruptcy risk при downturn |

## 11. Модель управления и стоимости принятия решений

Governance влияет на модель по четырем каналам: снижает effective employees через governance hours; увеличивает cash costs; замедляет решения через decision delay; меняет quality через decision quality multiplier. Для `traditional_company` положительное качество не предполагается автоматически. Для `worker_cooperative` высокая participation не создает uplift в `no_effect`.

## 12. Рыночная среда и случайные процессы

Все внешние процессы генерируются один раз на `run` и переиспользуются всеми сценариями при `common_random_numbers=true`.

| Stream | Использование |
|---|---|
| `market_stream` | market factor |
| `shock_stream` | Bernoulli shocks |
| `shock_cost_stream` | shock cost |
| `labor_stream` | stochastic turnover, если включен |
| `credit_stream` | stochastic credit market factor, если включен |

## 13. Денежные очереди и обязательства

| Queue | Что хранит | Cash effect |
|---|---|---|
| `ar_queue` | дебиторка к сбору | cash inflow in due month |
| `tax_payable_queue` | начисленный profit tax | cash outflow in due month |
| `employee_distribution_payable_queue` | начисленные выплаты сотрудникам | cash restricted at accrual, cash outflow at payment |
| `member_capital_redemption_queue` | выплаты внутренних счетов при уходе | cash outflow in due month |
| `capacity_addition_queue` | будущий ввод мощности | capacity increase in due month |

Priority cash payments: payroll, fixed costs, variable costs, workforce costs, governance cash costs, shock costs, taxes due, employee distributions due, member capital redemptions due, interest, principal, then new discretionary allocations.

## 14. Риски и условия банкротства

| Flag | Формула |
|---|---|
| `reserve_breach_flag` | `unrestricted_cash_close_t < required_cash_reserve_t` |
| `cash_gap_flag` | `cash_total_close_t < 0` |
| `liquidity_deficit_flag` | `unpaid_mandatory_obligations_t > epsilon` |
| `credit_limit_breach_flag` | `debt_balance_close_t > credit_line_limit_t + epsilon` |
| `bankruptcy_flag` | `liquidity_deficit_flag OR credit_limit_breach_flag` |
| `employee_distribution_cut_flag` | raw distribution > actual distribution |
| `reinvestment_underfunded_flag` | raw reinvestment > actual reinvestment total |

After bankruptcy and if `stop_after_bankruptcy=true`, the scenario becomes inactive, generates no revenue and performs no hiring.

## 15. Выходные показатели и формат отчета

### 15.1. Company metrics

| Метрика | Формула |
|---|---|
| `monthly_revenue` | `revenue_t` |
| `cumulative_revenue` | `sum_t revenue_t` |
| `operating_profit_before_allocation` | `revenue_t - operating_costs_before_allocation_t` |
| `net_profit_after_tax_and_employee_distribution` | see tax formula |
| `productive_capacity_revenue_monthly` | closing capacity |
| `capacity_growth_rate` | `capacity_close / capacity_start - 1` |
| `final_headcount` | `headcount_end_T` |
| `revenue_cagr` | `(revenue_last_12m / revenue_first_12m)^(12/(T-12)) - 1` |
| `capacity_cagr` | `(capacity_T / capacity_0)^(12/T) - 1` |
| `cash_end_total` | `cash_total_close_T` |
| `cash_end_unrestricted` | `unrestricted_cash_close_T` |
| `min_unrestricted_cash` | `min_t unrestricted_cash_close_t` |
| `liquidity_deficit_probability` | share of runs with any liquidity deficit |
| `bankruptcy_probability` | share of runs with any bankruptcy |
| `shock_survival_rate` | share of shock runs with no bankruptcy within 12 months after shock |

### 15.2. Workforce and employee metrics

| Метрика | Формула |
|---|---|
| `productivity_per_employee` | `revenue_t / max(epsilon, paid_employees_t)` |
| `turnover_rate_annual_average` | average monthly annualized turnover |
| `voluntary_leavers_total` | `sum_t voluntary_leavers_t` |
| `layoffs_total` | `sum_t distress_layoffs_t` |
| `hires_total` | `sum_t hires_t` |
| `hiring_and_onboarding_costs_total` | `sum_t hiring_cost_t` |
| `average_employee_income_monthly` | `(salary_paid + distribution_paid) / average_headcount` |
| `employee_cash_distribution_total` | `sum_t employee_cash_distribution_paid_t` |
| `member_capital_accounts_total` | closing member capital |
| `employee_income_volatility` | std dev monthly employee income |
| `risk_adjusted_employee_income` | `average_income - volatility_penalty_lambda * income_volatility` |
| `employee_risk_concentration_index_average` | average over months |

### 15.3. Development metrics

```text
reinvestment_total_cash = sum_t cash_reinvestment_paid_t
external_growth_capital_total = sum_t external_growth_capital_draw_t
actual_reinvestment_total = sum_t actual_reinvestment_total_t
reinvestment_underfunding_rate = sum(raw_reinvestment - actual_reinvestment_total) / sum(raw_reinvestment)
productive_capacity_added_total = sum_t capacity_added_by_investment_t
capacity_replacement_cost_proxy = productive_capacity_revenue_monthly / max(epsilon, capacity_revenue_created_per_currency_invested)
sustainable_development_value_proxy = cash_end_unrestricted + capacity_replacement_cost_proxy - debt_balance - unpaid_obligations - member_capital_redemption_due
```

`Sustainable_development_value_proxy` не является стоимостью доли основателя.

### 15.4. Summary output fields

```text
scenario, system_type, behavior_case, market_case, horizon_months,
median_cumulative_revenue, p10_cumulative_revenue,
median_operating_profit, median_net_profit,
median_productivity_per_employee, turnover_rate_annual_average,
hiring_and_onboarding_costs_total, average_employee_income_monthly,
risk_adjusted_employee_income, employee_cash_distribution_total,
member_capital_accounts_total, reinvestment_total_cash,
productive_capacity_growth_rate, cash_end_total_median,
cash_end_unrestricted_p10, min_unrestricted_cash_p10,
liquidity_deficit_probability, bankruptcy_probability,
shock_survival_rate, final_headcount_median,
revenue_cagr_median, capacity_cagr_median,
employee_risk_concentration_index_average, classification, assumption_flags
```

### 15.5. Monthly CSV fields

```text
run, month, scenario, system_type, behavior_case, market_case, active_company_flag,
market_factor, shock_happened, effective_collection_rate, headcount_begin,
voluntary_leavers, layoffs, hires, headcount_end, effective_employees,
productivity_uplift, productivity_multiplier, governance_hours,
governance_admin_equivalent_employees, governance_cash_cost, decision_delay_months,
decision_quality_multiplier, fairness_index, free_rider_penalty,
employee_risk_concentration_index, market_demand, labor_revenue_capacity,
productive_capacity_revenue_monthly, revenue, salary_cost, fixed_costs,
variable_costs, turnover_and_workforce_cost, shock_cost,
operating_profit_before_allocation, interest_expense,
profit_before_tax_before_distribution, positive_result_base,
cash_collected_current, cash_collected_from_ar, mandatory_cash_payments,
cash_after_mandatory, credit_draw_for_liquidity,
employee_cash_distribution_accrued, employee_cash_distribution_paid,
member_capital_allocation, member_capital_redemption_due,
reinvestment_cash_paid, external_growth_capital_draw,
organizational_reserve_allocation, profit_tax_accrual, taxes_paid,
cash_total_close, restricted_distribution_cash_close,
restricted_reserve_cash_close, unrestricted_cash_close, debt_balance_close,
required_cash_reserve, reserve_breach_flag, cash_gap_flag,
liquidity_deficit_flag, bankruptcy_flag
```

## 16. Парное сравнение сценариев

For each run `r`, scenario `s`, reference `b`:

```text
paired_delta_X[r,s,b] = X[r,s] - X[r,b]
```

Default reference is `traditional_company`. Additional reference `profit_sharing` isolates incremental ownership/governance effects. Report median, P10, P90, probability positive and probability negative. A scenario cannot be classified as development-dominant if bankruptcy or liquidity deficit probability is worse than reference by more than tolerance.

## 17. Monte Carlo и sensitivity analysis

Minimum Monte Carlo: `runs>=1000`, `common_random_numbers=true`, horizons include 60 and 120 months. Publication-quality stress report should use `runs>=10000`.

Mandatory sensitivity grid includes productivity uplift, ownership productivity sensitivity, retention effect, governance intensity/hours, decision quality, free rider, fairness, risk concentration, external capital access, reinvestment rate, cash reserve, shock probability, shock severity and collection rate.

Break-even motivation effect:

```text
BE_productivity_uplift_s = inf { u |
  median(paired_delta_sustainable_development_value_proxy(u)) >= 0
  AND bankruptcy_probability_s(u) <= bankruptcy_probability_b + tolerance
  AND liquidity_deficit_probability_s(u) <= liquidity_deficit_probability_b + tolerance
}
```

If no value exists inside `break_even_uplift_range`, return null and flag `no_break_even_in_tested_range`.

## 18. Проверяемые инварианты

| Инвариант | Формула |
|---|---|
| Headcount identity | `headcount_end = headcount_begin - voluntary_leavers - layoffs + hires` |
| Non-negative headcount | `headcount_end >= -epsilon` |
| Revenue bounds | `revenue <= market_demand`, `revenue <= labor_revenue_capacity`, `revenue <= productive_capacity_limit` |
| Cash identity | `cash_close = cash_begin + collections + credit_draws + external_capital_inflows - mandatory_payments - reinvestment_cash - external_distribution - external_capital_spent` |
| Restricted cash identity | `restricted_cash = restricted_distribution_cash + restricted_reserve_cash` |
| Unrestricted cash identity | `unrestricted_cash = cash_total - restricted_cash` |
| Debt identity | `debt_close = debt_begin - principal_paid + credit_draw_for_liquidity + external_growth_debt_draw` |
| Capacity identity | `capacity_close = capacity_begin * (1 - depreciation) + capacity_additions_due` |
| Member capital identity | `member_capital_close = member_capital_begin + allocation - redemption_accrual` |
| Allocation sum | `sum(policy rates) <= 1` |
| Cash-safe allocation | `sum(actual_allocations * cash_multiplier) <= cash_safe_allocation_budget + epsilon` |
| No automatic motivation | In `no_effect`, structural ownership/profit/governance cannot create productivity/retention effect |
| Bankruptcy absorbing | If bankruptcy at `t`, active flag false for all future months when enabled |

## 19. Валидация конфигурации

Strict validation must reject unknown fields, duplicate JSON fields, NaN/Infinity, missing required fields, invalid enums, negative money where disallowed, rates outside [0,1], horizons greater than months, duplicate scenario names, unknown behavior/scenario refs, invalid allocation priority items and allocation rate sums above 1.

Normalization rules: annual rates are not overwritten; monthly equivalents are derived. Null caps mean no cap. Empty `behavior_case_refs` is an error. In deterministic mode `runs` must be `1`. `ramp_productivity_multipliers.length` must equal `ramp_duration_months` unless duration is 0.

Warnings: possible turnover double count, inconsistent scenario label, high redemption liquidity risk, external distribution not being a success metric, simplified tax model.

## 20. Набор обязательных тестов и эталонных примеров

The canonical test manifest is provided in `required_tests_v0_4.json`. Required tests include baseline sanity, headcount identity, no-effect control, profit sharing without ownership, governance cost, cash-safe distribution, distribution reserve, member capital redemption queue, reinvestment capacity lag, external capital constraint, common random numbers, shock response quality, bankruptcy absorbing behavior, strict unknown-field rejection, duplicate JSON key rejection and v0.3 compatibility tests.

## 21. Ограничения модели и недопустимые интерпретации

The model must not be interpreted as proof that one organizational system is universally better. Behavioral effects are assumptions. The workforce is aggregated. Tax logic is simplified. The model is not a full balance sheet. Member capital is simplified. Governance is stylized. Market paths are stress tools. External capital is simplified. Sustainable development value proxy is not equity value.

Недопустимые выводы: `worker_cooperative` всегда лучше; `traditional_company` всегда эффективнее; motivation effect доказан моделью; модель оценила стоимость доли основателя; результат можно применять без sensitivity analysis.

## 22. План миграции с v0.3

Сохраняются: monthly discrete simulation, deterministic/Monte Carlo, random seed, common random numbers, paired deltas, P&L/cash separation, AR queue, tax payable queue, employee distribution payable queue, restricted cash at accrual, cash-safe distribution base, behavior cases, no-effect and negative controls, sensitivity outputs, monthly CSV, validation and sanity tests.

Deprecated as primary metrics: `owner_distributable_cash`, `owner_dividend_policy`, `fixed_raise_same_expected_cost` as main comparator, constant `employees_count`, `demand_cap_multiplier` as sole capacity constraint.

New modules: `WorkforceDynamics`, `ProductionCapacity`, `GovernanceModel`, `OwnershipAndMemberCapital`, `RiskConcentration`, `ExternalCapitalAccess`, `ResultAllocationPolicy`, `ShockResponse`, `DevelopmentMetrics`, `CompatibilityV03`.

`profit_share_equal_10` maps to `profit_sharing` with ownership=0, governance=0, cash distribution=0.10, member capital=0, distribution rule `equal_per_capita`, period 1 month, payout lag 1 month.

Compatibility mode must reproduce `fixed_only`, profit share 5/10/15, above-hurdle variants, monthly/quarterly/annual periods, v0.3 cash-safe accrual, restricted cash at accrual, AR/tax/bonus queues and v0.3 risk flags.

See `migration_map_v0_3_to_v0_4.csv` for field-level mapping.

## 23. Критерии готовности реализации

Implementation is ready when: strict validation passes; all organizational scenarios are implemented; ownership, distribution and governance are independent; no-effect does not create uplift; headcount and cash identities hold; P&L and cash are separated; distributions are cash-safe and restricted at accrual; reinvestment is cash outflow that creates capacity after lag; governance costs/delay/quality work by formula; free rider/fairness/risk concentration are explicit; external capital constraints affect liquidity and growth; shocks use common random numbers; paired deltas are calculated per run; Monte Carlo is reproducible by seed; deterministic mode passes sanity tests; sensitivity and break-even analyses work; v0.3 compatibility tests pass; reports exclude founder-share value as a target metric; limitations and assumption flags are printed.

# Appendix A. Architecture

| Модуль | Ответственность |
|---|---|
| `SimulationEngine` | runs, seed, common random numbers, horizons, scenario matrix |
| `ScenarioExpander` | matrix `organizational_scenario x behavior_case x market_case` |
| `EnvironmentPath` | рынок, инфляция, шоки, трудовой рынок, кредитный рынок |
| `WorkforceDynamics` | headcount, hires, leavers, layoffs, ramp-up, текучесть |
| `BehaviorMechanisms` | motivation assumptions, fairness, free rider, risk concentration, retention |
| `GovernanceModel` | governance hours, cost, delay, decision quality |
| `ProductionModel` | labor capacity, productive capacity, demand, revenue |
| `CompanyEconomics` | revenue, costs, operating profit, net profit |
| `ResultAllocationPolicy` | distribution, reinvestment, reserves, member capital |
| `CashFlowModel` | cash, restricted cash, AR, taxes, distributions, debt |
| `FinancingModel` | credit line, liquidity draw, external growth capital |
| `Metrics` | monthly, cumulative, paired deltas, percentiles, risk metrics |
| `SensitivityRunner` | grids, tornado, break-even motivation effect |
| `Validation` | config validation, invariants, unit and sanity tests |
| `CompatibilityV03` | v0.3 reproduction mode |


# Appendix B. Full JSON configuration example

```json
{
  "schema_version": "0.4",
  "config_validation": {
    "reject_unknown_fields": true,
    "reject_duplicate_fields": true,
    "allow_name_normalization": false,
    "reject_nan_and_infinity": true
  },
  "simulation": {
    "mode": "monte_carlo",
    "months": 240,
    "horizons_months": [
      60,
      120,
      240
    ],
    "runs": 1000,
    "random_seed": 42,
    "common_random_numbers": true,
    "currency": "RUB",
    "epsilon": 1e-09,
    "headcount_mode": "fractional",
    "stop_after_bankruptcy": true
  },
  "units": {
    "money": "RUB nominal unless simulation.currency is changed",
    "rates": "decimal share, 0.10 means 10 percent",
    "percentage_points": "absolute annual rate change, -0.03 means -3 pp",
    "time_step": "one month",
    "company_economics.initial_headcount": "people",
    "company_economics.base_salary_per_employee_monthly": "RUB per person per month",
    "company_economics.base_revenue_per_effective_employee_monthly": "RUB revenue per effective employee per month",
    "company_economics.initial_market_demand_monthly": "RUB revenue demand per month",
    "company_economics.initial_productive_capacity_revenue_monthly": "RUB revenue capacity per month",
    "company_economics.fixed_costs_monthly": "RUB per month",
    "company_economics.capacity_revenue_created_per_currency_invested": "RUB monthly revenue capacity per RUB invested",
    "market.market_growth_monthly": "monthly rate",
    "market.market_volatility_monthly": "monthly standard deviation",
    "workforce.base_turnover_rate_annual": "annual share",
    "workforce.ramp_productivity_multipliers": "productivity multiplier by tenure month",
    "employee_risk.employee_external_savings_proxy_per_employee": "RUB per employee",
    "financing.base_credit_line": "RUB",
    "organizational_scenarios.employee_cash_distribution_rate": "share of positive result base",
    "organizational_scenarios.member_capital_allocation_rate": "share of positive result base",
    "organizational_scenarios.reinvestment_rate": "share of positive result base",
    "organizational_scenarios.governance.governance_participation_intensity": "0 to 1 index"
  },
  "company_economics": {
    "initial_headcount": 50,
    "base_salary_per_employee_monthly": 100000,
    "salary_payroll_tax_rate": 0.0,
    "standard_hours_per_employee_month": 160,
    "base_revenue_per_effective_employee_monthly": 230000,
    "initial_market_demand_monthly": 11500000,
    "initial_productive_capacity_revenue_monthly": 13800000,
    "fixed_costs_monthly": 2000000,
    "variable_cost_rate": 0.25,
    "profit_tax_rate": 0.2,
    "profit_tax_payment_lag_months": 1,
    "cost_inflation_monthly": 0.005,
    "capacity_depreciation_rate_monthly": 0.002,
    "capacity_revenue_created_per_currency_invested": 0.08,
    "investment_activation_lag_months": 3,
    "required_cash_reserve_months": 2.0,
    "starting_cash": 15000000,
    "opening_accounts_receivable": 1725000
  },
  "market": {
    "market_process": "bounded_lognormal",
    "market_growth_monthly": 0.003,
    "market_volatility_monthly": 0.08,
    "market_factor_min": 0.5,
    "market_factor_max": 1.8,
    "seasonality_multipliers": [
      1.0,
      1.0,
      1.0,
      1.0,
      1.0,
      1.0,
      1.0,
      1.0,
      1.0,
      1.0,
      1.0,
      1.0
    ],
    "revenue_collection_rate_current_month": 0.85,
    "accounts_receivable_lag_months": 1,
    "bad_debt_rate": 0.0,
    "shock_probability_monthly": 0.03,
    "shock_revenue_multiplier": 0.8,
    "shock_cost_mean": 0,
    "shock_cost_std": 0,
    "cash_collection_stress_multiplier": 0.9,
    "labor_market_factor": 1.0,
    "credit_market_factor": 1.0
  },
  "workforce": {
    "base_turnover_rate_annual": 0.2,
    "min_turnover_rate_annual": 0.03,
    "max_turnover_rate_annual": 0.6,
    "turnover_random_mode": "deterministic",
    "high_performer_share": 0.2,
    "recruiting_cost_per_hire": 50000,
    "onboarding_cost_per_hire": 50000,
    "manager_time_cost_per_hire": 25000,
    "exit_admin_cost_per_leaver": 10000,
    "lost_productivity_cost_per_leaver": 0,
    "severance_cost_per_layoff": 100000,
    "ramp_duration_months": 3,
    "ramp_productivity_multipliers": [
      0.5,
      0.75,
      0.9
    ],
    "max_hires_per_month_rate": 0.1,
    "max_layoffs_per_month_rate": 0.1,
    "layoff_trigger_cash_ratio": 0.5,
    "leaver_paid_fraction_of_month": 0.5,
    "new_hire_paid_fraction_of_month": 0.5,
    "min_productivity_uplift": -0.15,
    "max_productivity_uplift": 0.2,
    "turnover_productivity_penalty_per_annual_turnover": 0.1,
    "target_staffing_buffer": 1.05,
    "max_cash_share_for_hiring": 0.25
  },
  "employee_risk": {
    "employee_external_savings_proxy_per_employee": 600000,
    "employment_dependence_index": 1.0,
    "risk_weight_variable_income": 0.4,
    "risk_weight_member_capital": 0.4,
    "risk_weight_employment_dependence": 0.2
  },
  "financing": {
    "base_credit_line": 0,
    "debt_interest_rate_annual": 0.18,
    "scheduled_principal_payment_monthly": 0,
    "external_growth_capital_limit_monthly": 0,
    "external_capital_type": "debt",
    "distribution_payroll_tax_rate": 0.0,
    "distribution_tax_deductible_share": 1.0,
    "employee_distribution_payout_lag_months": 1,
    "member_capital_redemption_lag_months": 24,
    "member_capital_redemption_fraction_on_exit": 1.0,
    "reserve_release_rate_on_stress": 0.5
  },
  "behavior_cases": {
    "no_effect": {
      "base_productivity_uplift_direct": 0.0,
      "ownership_productivity_sensitivity": 0.0,
      "profit_distribution_productivity_sensitivity": 0.0,
      "governance_voice_productivity_sensitivity": 0.0,
      "base_turnover_delta_annual_pp": 0.0,
      "ownership_retention_delta_annual_pp_per_full_ownership": 0.0,
      "profit_distribution_retention_delta_annual_pp_per_10pp": 0.0,
      "governance_retention_delta_annual_pp_per_full_participation": 0.0,
      "fairness_base": 0.0,
      "transparency_to_fairness": 0.0,
      "equal_distribution_fairness_effect": 0.0,
      "contribution_based_distribution_fairness_effect": 0.0,
      "pay_dispersion_fairness_penalty": 0.0,
      "unpaid_governance_burden_penalty": 0.0,
      "zero_distribution_fairness_penalty": 0.0,
      "fairness_productivity_sensitivity": 0.0,
      "fairness_turnover_sensitivity_annual_pp": 0.0,
      "free_rider_base_penalty": 0.0,
      "free_rider_size_exponent": 0.5,
      "free_rider_reference_headcount": 50,
      "free_rider_max_size_multiplier": 3.0,
      "risk_concentration_turnover_sensitivity_annual_pp": 0.0,
      "income_volatility_turnover_sensitivity_annual_pp": 0.0,
      "high_performer_attrition_delta_pp": 0.0
    },
    "retention_only": {
      "base_productivity_uplift_direct": 0.0,
      "ownership_productivity_sensitivity": 0.0,
      "profit_distribution_productivity_sensitivity": 0.0,
      "governance_voice_productivity_sensitivity": 0.0,
      "base_turnover_delta_annual_pp": 0.0,
      "ownership_retention_delta_annual_pp_per_full_ownership": -0.04,
      "profit_distribution_retention_delta_annual_pp_per_10pp": -0.01,
      "governance_retention_delta_annual_pp_per_full_participation": -0.01,
      "fairness_base": 0.0,
      "transparency_to_fairness": 0.1,
      "equal_distribution_fairness_effect": 0.0,
      "contribution_based_distribution_fairness_effect": 0.0,
      "pay_dispersion_fairness_penalty": 0.0,
      "unpaid_governance_burden_penalty": 0.0,
      "zero_distribution_fairness_penalty": 0.0,
      "fairness_productivity_sensitivity": 0.0,
      "fairness_turnover_sensitivity_annual_pp": 0.02,
      "free_rider_base_penalty": 0.0,
      "free_rider_size_exponent": 0.5,
      "free_rider_reference_headcount": 50,
      "free_rider_max_size_multiplier": 3.0,
      "risk_concentration_turnover_sensitivity_annual_pp": 0.0,
      "income_volatility_turnover_sensitivity_annual_pp": 0.0,
      "high_performer_attrition_delta_pp": 0.0
    },
    "moderate_positive": {
      "base_productivity_uplift_direct": 0.0,
      "ownership_productivity_sensitivity": 0.03,
      "profit_distribution_productivity_sensitivity": 0.005,
      "governance_voice_productivity_sensitivity": 0.01,
      "base_turnover_delta_annual_pp": 0.0,
      "ownership_retention_delta_annual_pp_per_full_ownership": -0.04,
      "profit_distribution_retention_delta_annual_pp_per_10pp": -0.01,
      "governance_retention_delta_annual_pp_per_full_participation": -0.01,
      "fairness_base": 0.0,
      "transparency_to_fairness": 0.2,
      "equal_distribution_fairness_effect": 0.02,
      "contribution_based_distribution_fairness_effect": 0.02,
      "pay_dispersion_fairness_penalty": 0.0,
      "unpaid_governance_burden_penalty": 0.05,
      "zero_distribution_fairness_penalty": 0.02,
      "fairness_productivity_sensitivity": 0.02,
      "fairness_turnover_sensitivity_annual_pp": 0.02,
      "free_rider_base_penalty": 0.005,
      "free_rider_size_exponent": 0.5,
      "free_rider_reference_headcount": 50,
      "free_rider_max_size_multiplier": 3.0,
      "risk_concentration_turnover_sensitivity_annual_pp": 0.01,
      "income_volatility_turnover_sensitivity_annual_pp": 0.01,
      "high_performer_attrition_delta_pp": 0.0
    },
    "governance_costly": {
      "base_productivity_uplift_direct": 0.0,
      "ownership_productivity_sensitivity": 0.0,
      "profit_distribution_productivity_sensitivity": 0.0,
      "governance_voice_productivity_sensitivity": 0.0,
      "base_turnover_delta_annual_pp": 0.0,
      "ownership_retention_delta_annual_pp_per_full_ownership": 0.0,
      "profit_distribution_retention_delta_annual_pp_per_10pp": 0.0,
      "governance_retention_delta_annual_pp_per_full_participation": 0.0,
      "fairness_base": 0.0,
      "transparency_to_fairness": 0.0,
      "equal_distribution_fairness_effect": 0.0,
      "contribution_based_distribution_fairness_effect": 0.0,
      "pay_dispersion_fairness_penalty": 0.0,
      "unpaid_governance_burden_penalty": 0.1,
      "zero_distribution_fairness_penalty": 0.0,
      "fairness_productivity_sensitivity": 0.0,
      "fairness_turnover_sensitivity_annual_pp": 0.0,
      "free_rider_base_penalty": 0.0,
      "free_rider_size_exponent": 0.5,
      "free_rider_reference_headcount": 50,
      "free_rider_max_size_multiplier": 3.0,
      "risk_concentration_turnover_sensitivity_annual_pp": 0.0,
      "income_volatility_turnover_sensitivity_annual_pp": 0.0,
      "high_performer_attrition_delta_pp": 0.0
    },
    "negative_fairness_free_rider": {
      "base_productivity_uplift_direct": -0.01,
      "ownership_productivity_sensitivity": 0.0,
      "profit_distribution_productivity_sensitivity": 0.0,
      "governance_voice_productivity_sensitivity": 0.0,
      "base_turnover_delta_annual_pp": 0.02,
      "ownership_retention_delta_annual_pp_per_full_ownership": 0.0,
      "profit_distribution_retention_delta_annual_pp_per_10pp": 0.0,
      "governance_retention_delta_annual_pp_per_full_participation": 0.0,
      "fairness_base": -0.1,
      "transparency_to_fairness": 0.0,
      "equal_distribution_fairness_effect": -0.15,
      "contribution_based_distribution_fairness_effect": 0.02,
      "pay_dispersion_fairness_penalty": 0.1,
      "unpaid_governance_burden_penalty": 0.1,
      "zero_distribution_fairness_penalty": 0.05,
      "fairness_productivity_sensitivity": 0.03,
      "fairness_turnover_sensitivity_annual_pp": 0.04,
      "free_rider_base_penalty": 0.03,
      "free_rider_size_exponent": 0.5,
      "free_rider_reference_headcount": 50,
      "free_rider_max_size_multiplier": 3.0,
      "risk_concentration_turnover_sensitivity_annual_pp": 0.02,
      "income_volatility_turnover_sensitivity_annual_pp": 0.02,
      "high_performer_attrition_delta_pp": 0.05
    },
    "risk_concentration_negative": {
      "base_productivity_uplift_direct": 0.0,
      "ownership_productivity_sensitivity": 0.0,
      "profit_distribution_productivity_sensitivity": 0.0,
      "governance_voice_productivity_sensitivity": 0.0,
      "base_turnover_delta_annual_pp": 0.0,
      "ownership_retention_delta_annual_pp_per_full_ownership": -0.02,
      "profit_distribution_retention_delta_annual_pp_per_10pp": 0.0,
      "governance_retention_delta_annual_pp_per_full_participation": 0.0,
      "fairness_base": 0.0,
      "transparency_to_fairness": 0.0,
      "equal_distribution_fairness_effect": 0.0,
      "contribution_based_distribution_fairness_effect": 0.0,
      "pay_dispersion_fairness_penalty": 0.0,
      "unpaid_governance_burden_penalty": 0.0,
      "zero_distribution_fairness_penalty": 0.0,
      "fairness_productivity_sensitivity": 0.0,
      "fairness_turnover_sensitivity_annual_pp": 0.0,
      "free_rider_base_penalty": 0.0,
      "free_rider_size_exponent": 0.5,
      "free_rider_reference_headcount": 50,
      "free_rider_max_size_multiplier": 3.0,
      "risk_concentration_turnover_sensitivity_annual_pp": 0.08,
      "income_volatility_turnover_sensitivity_annual_pp": 0.03,
      "high_performer_attrition_delta_pp": 0.02
    }
  },
  "organizational_scenarios": [
    {
      "name": "traditional_company",
      "system_type": "traditional_company",
      "employee_ownership_fraction": 0.0,
      "employee_cash_distribution_rate": 0.0,
      "member_capital_allocation_rate": 0.0,
      "reinvestment_rate": 0.15,
      "organizational_reserve_rate": 0.05,
      "external_distribution_rate": 0.0,
      "result_hurdle_monthly": 0,
      "allocation_priority": [
        "reinvestment",
        "organizational_reserve"
      ],
      "distribution_rule": "none",
      "contribution_measurement_quality": 0.5,
      "peer_monitoring_effectiveness": 0.0,
      "transparency_index": 0.3,
      "employment_stabilization_preference": 0.2,
      "external_capital_access_multiplier": 1.0,
      "profit_distribution_period_months": 1,
      "max_distribution_per_employee_period": null,
      "governance": {
        "governance_participation_intensity": 0.0,
        "base_governance_hours_per_employee_month": 0.0,
        "fixed_governance_hours_monthly": 0.0,
        "governance_cash_cost_fixed_monthly": 0,
        "governance_cash_cost_per_employee_monthly": 0,
        "decision_complexity_index": 1.0,
        "base_decision_delay_months": 0.25,
        "delay_per_participation_months": 0.0,
        "local_autonomy_index": 0.2,
        "decentralization_speed_gain_months": 0.0,
        "governance_capability_index": 1.0,
        "information_sharing_quality": 0.4,
        "trust_index": 0.5,
        "quality_gain_from_participation": 0.0,
        "coordination_loss_from_participation": 0.0,
        "conflict_loss_sensitivity": 0.0,
        "decision_delay_quality_loss": 0.0,
        "decision_quality_min": 0.7,
        "decision_quality_max": 1.2,
        "shock_mitigation_sensitivity": 0.0,
        "shock_delay_amplification": 0.0,
        "investment_efficiency_sensitivity": 0.0
      },
      "behavior_case_refs": [
        "no_effect",
        "retention_only",
        "moderate_positive",
        "governance_costly",
        "negative_fairness_free_rider",
        "risk_concentration_negative"
      ]
    },
    {
      "name": "profit_sharing",
      "system_type": "profit_sharing",
      "employee_ownership_fraction": 0.0,
      "employee_cash_distribution_rate": 0.1,
      "member_capital_allocation_rate": 0.0,
      "reinvestment_rate": 0.1,
      "organizational_reserve_rate": 0.05,
      "external_distribution_rate": 0.0,
      "result_hurdle_monthly": 0,
      "allocation_priority": [
        "organizational_reserve",
        "reinvestment",
        "employee_cash_distribution"
      ],
      "distribution_rule": "equal_per_capita",
      "contribution_measurement_quality": 0.4,
      "peer_monitoring_effectiveness": 0.2,
      "transparency_index": 0.6,
      "employment_stabilization_preference": 0.2,
      "external_capital_access_multiplier": 1.0,
      "profit_distribution_period_months": 1,
      "max_distribution_per_employee_period": null,
      "governance": {
        "governance_participation_intensity": 0.0,
        "base_governance_hours_per_employee_month": 0.0,
        "fixed_governance_hours_monthly": 0.0,
        "governance_cash_cost_fixed_monthly": 0,
        "governance_cash_cost_per_employee_monthly": 0,
        "decision_complexity_index": 1.0,
        "base_decision_delay_months": 0.25,
        "delay_per_participation_months": 0.0,
        "local_autonomy_index": 0.2,
        "decentralization_speed_gain_months": 0.0,
        "governance_capability_index": 1.0,
        "information_sharing_quality": 0.5,
        "trust_index": 0.5,
        "quality_gain_from_participation": 0.0,
        "coordination_loss_from_participation": 0.0,
        "conflict_loss_sensitivity": 0.0,
        "decision_delay_quality_loss": 0.0,
        "decision_quality_min": 0.7,
        "decision_quality_max": 1.2,
        "shock_mitigation_sensitivity": 0.0,
        "shock_delay_amplification": 0.0,
        "investment_efficiency_sensitivity": 0.0
      },
      "behavior_case_refs": [
        "no_effect",
        "retention_only",
        "moderate_positive",
        "negative_fairness_free_rider",
        "risk_concentration_negative"
      ]
    },
    {
      "name": "employee_ownership_partial",
      "system_type": "employee_ownership_partial",
      "employee_ownership_fraction": 0.3,
      "employee_cash_distribution_rate": 0.1,
      "member_capital_allocation_rate": 0.05,
      "reinvestment_rate": 0.15,
      "organizational_reserve_rate": 0.05,
      "external_distribution_rate": 0.0,
      "result_hurdle_monthly": 0,
      "allocation_priority": [
        "organizational_reserve",
        "reinvestment",
        "employee_cash_distribution",
        "member_capital_allocation"
      ],
      "distribution_rule": "hybrid",
      "contribution_measurement_quality": 0.6,
      "peer_monitoring_effectiveness": 0.4,
      "transparency_index": 0.7,
      "employment_stabilization_preference": 0.5,
      "external_capital_access_multiplier": 0.8,
      "profit_distribution_period_months": 1,
      "max_distribution_per_employee_period": null,
      "governance": {
        "governance_participation_intensity": 0.3,
        "base_governance_hours_per_employee_month": 2.0,
        "fixed_governance_hours_monthly": 20.0,
        "governance_cash_cost_fixed_monthly": 50000,
        "governance_cash_cost_per_employee_monthly": 1000,
        "decision_complexity_index": 1.0,
        "base_decision_delay_months": 0.25,
        "delay_per_participation_months": 0.4,
        "local_autonomy_index": 0.4,
        "decentralization_speed_gain_months": 0.1,
        "governance_capability_index": 1.0,
        "information_sharing_quality": 0.7,
        "trust_index": 0.6,
        "quality_gain_from_participation": 0.02,
        "coordination_loss_from_participation": 0.01,
        "conflict_loss_sensitivity": 0.01,
        "decision_delay_quality_loss": 0.0,
        "decision_quality_min": 0.7,
        "decision_quality_max": 1.2,
        "shock_mitigation_sensitivity": 0.1,
        "shock_delay_amplification": 0.02,
        "investment_efficiency_sensitivity": 0.1
      },
      "behavior_case_refs": [
        "no_effect",
        "retention_only",
        "moderate_positive",
        "governance_costly",
        "negative_fairness_free_rider",
        "risk_concentration_negative"
      ]
    },
    {
      "name": "worker_cooperative",
      "system_type": "worker_cooperative",
      "employee_ownership_fraction": 1.0,
      "employee_cash_distribution_rate": 0.15,
      "member_capital_allocation_rate": 0.05,
      "reinvestment_rate": 0.25,
      "organizational_reserve_rate": 0.1,
      "external_distribution_rate": 0.0,
      "result_hurdle_monthly": 0,
      "allocation_priority": [
        "organizational_reserve",
        "reinvestment",
        "employee_cash_distribution",
        "member_capital_allocation"
      ],
      "distribution_rule": "equal_per_capita",
      "contribution_measurement_quality": 0.5,
      "peer_monitoring_effectiveness": 0.6,
      "transparency_index": 0.8,
      "employment_stabilization_preference": 0.8,
      "external_capital_access_multiplier": 0.5,
      "profit_distribution_period_months": 1,
      "max_distribution_per_employee_period": null,
      "governance": {
        "governance_participation_intensity": 0.8,
        "base_governance_hours_per_employee_month": 4.0,
        "fixed_governance_hours_monthly": 40.0,
        "governance_cash_cost_fixed_monthly": 100000,
        "governance_cash_cost_per_employee_monthly": 2000,
        "decision_complexity_index": 1.2,
        "base_decision_delay_months": 0.25,
        "delay_per_participation_months": 0.5,
        "local_autonomy_index": 0.6,
        "decentralization_speed_gain_months": 0.2,
        "governance_capability_index": 1.0,
        "information_sharing_quality": 0.8,
        "trust_index": 0.7,
        "quality_gain_from_participation": 0.03,
        "coordination_loss_from_participation": 0.02,
        "conflict_loss_sensitivity": 0.02,
        "decision_delay_quality_loss": 0.01,
        "decision_quality_min": 0.7,
        "decision_quality_max": 1.2,
        "shock_mitigation_sensitivity": 0.15,
        "shock_delay_amplification": 0.03,
        "investment_efficiency_sensitivity": 0.15
      },
      "behavior_case_refs": [
        "no_effect",
        "retention_only",
        "moderate_positive",
        "governance_costly",
        "negative_fairness_free_rider",
        "risk_concentration_negative"
      ]
    }
  ],
  "analysis": {
    "paired_reference_scenarios": [
      "traditional_company",
      "profit_sharing"
    ],
    "volatility_penalty_lambda": 0.25,
    "classification_tolerance": 0.01,
    "break_even_metric": "sustainable_development_value_proxy",
    "break_even_uplift_range": [
      -0.05,
      0.15
    ],
    "sensitivity_parameters": [
      {
        "path": "behavior_cases.moderate_positive.ownership_productivity_sensitivity",
        "values": [
          -0.05,
          0.0,
          0.02,
          0.05,
          0.1
        ]
      },
      {
        "path": "behavior_cases.moderate_positive.ownership_retention_delta_annual_pp_per_full_ownership",
        "values": [
          0.05,
          0.0,
          -0.02,
          -0.05,
          -0.1
        ]
      },
      {
        "path": "organizational_scenarios.worker_cooperative.governance.base_governance_hours_per_employee_month",
        "values": [
          0,
          1,
          4,
          8,
          16
        ]
      },
      {
        "path": "behavior_cases.negative_fairness_free_rider.free_rider_base_penalty",
        "values": [
          0,
          0.01,
          0.03,
          0.07
        ]
      },
      {
        "path": "organizational_scenarios.worker_cooperative.external_capital_access_multiplier",
        "values": [
          0.25,
          0.5,
          0.75,
          1.0
        ]
      },
      {
        "path": "company_economics.required_cash_reserve_months",
        "values": [
          1,
          2,
          3,
          6
        ]
      },
      {
        "path": "market.revenue_collection_rate_current_month",
        "values": [
          0.7,
          0.85,
          1.0
        ]
      },
      {
        "path": "market.shock_revenue_multiplier",
        "values": [
          0.6,
          0.8,
          0.9
        ]
      }
    ]
  },
  "reporting": {
    "print_model_limitations": true,
    "print_assumption_flags": true,
    "write_monthly_csv": true,
    "write_summary_csv": true,
    "write_sensitivity_csv": true,
    "write_break_even_csv": true
  },
  "compatibility_v0_3": {
    "enabled": false,
    "preserve_v0_3_profit_sharing_scenarios": true,
    "legacy_outputs_enabled": false,
    "headcount_policy": "dynamic_v0_4"
  }
}
```

# Appendix C. Self-check

| Проверка | Статус |
|---|---|
| Нет ли противоречий между разделами? | Явных противоречий нет |
| Определены ли все используемые переменные? | Да: через parameters, state или formulas |
| Однозначен ли порядок расчета месяца? | Да: раздел 7 |
| Можно ли реализовать модель без дополнительных экономических решений? | Да: formulas, defaults, ranges and priorities are specified |
| Нейтральна ли модель? | Да: no-effect prevents automatic motivation effect |
| Можно ли проверить формулы тестами? | Да: invariants and required test manifest are specified |
