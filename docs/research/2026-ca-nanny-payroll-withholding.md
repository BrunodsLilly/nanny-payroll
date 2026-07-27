# 2026 Household-Employer ("Nanny Tax") Payroll Withholding — Single CA Filer, Weekly Pay

Research capture for building an accurate payroll tax engine (supersedes the flat-rate
placeholder in ADR-0005). All figures are for **tax year 2026** (the year in force as of the
research date below), pulled from IRS and California EDD primary sources. Each number is
cited to the owning document.

- **Situation modeled:** one household employee, single filing status, paid **weekly**,
  $27/hr × 40 hrs = **$1,080.00 gross/week** (≈ $56,160/yr). Federal W-4 is a plain post-2020
  form (single, Step 2 box unchecked, no dependents/adjustments); CA DE 4 claims **1** regular
  withholding allowance.
- **Research date:** 2026-07-24. **Tax year:** 2026.

> Not tax advice. These are the statutory withholding *methods*; a filer should confirm with a
> CPA or payroll provider.

## Summary of rates (2026)

| Item | Who pays | 2026 value | Basis |
|---|---|---|---|
| Social Security (OASDI) | employee + employer | 6.2% each, up to $184,500 wage base | IRS Pub 926 (2026) |
| Medicare (HI) | employee + employer | 1.45% each, no wage cap | IRS Pub 926 (2026) |
| Additional Medicare | employee | +0.9% on wages over $200,000 | IRS Pub 926 |
| Federal income tax (FIT) | employee (optional) | Pub 15-T percentage method | IRS Pub 15-T (2026) |
| CA SDI | employee | **1.3%**, no wage cap | CA EDD Rates & Withholding (2026) |
| CA income tax (PIT) | employee | DE 44 Method B schedules | CA EDD 2026 Method B |
| FUTA | employer | 6% on first $7,000, less 5.4% credit → **0.6% net** | IRS Pub 926 (2026) |
| CA UI | employer | new employer 3.4% (range 1.5–6.2%), first $7,000 | CA EDD (2026) |
| CA ETT | employer | 0.1%, first $7,000 | CA EDD (2026) |

Key coverage thresholds (2026): Social Security/Medicare apply once you pay a household
worker **$3,000+** in cash wages in the year; **FUTA** applies if you pay **$1,000+** in any
calendar quarter (Pub 926, 2026).

## 1. Social Security (OASDI)

- Employee rate **6.2%**, employer rate **6.2%**. Wage base limit for 2026 is **$184,500**
  (irrelevant at $56k/yr). *Source: IRS Pub 926 (2026), "Social Security and Medicare Taxes for 2026."*
- Weekly: `1080 × 0.062 = $66.96`.

## 2. Medicare (HI)

- Employee **1.45%**, employer **1.45%**, **no wage base limit**. *Source: IRS Pub 926 (2026).*
- Additional Medicare Tax of **0.9%** on wages over **$200,000** (employee only; employer does
  not match). Not reached here. *Source: IRS Pub 926.*
- Weekly: `1080 × 0.0145 = $15.66`.

## 3. CA State Disability Insurance (SDI)

- **SDI withholding rate for 2026 is 1.3%** (employee-paid). *Source: CA EDD, "Rates and
  Withholding" → SDI Rate.*
- **No taxable wage limit / no maximum withholding.** SB 951 removed the SDI taxable wage
  ceiling effective Jan 1, 2024, and it remains removed for 2026. *Source: same EDD page.*
- Rate history for reference: 2024 = 1.1%, 2025 = 1.2%, **2026 = 1.3%**.
  (The current code's `sdi = 0.009` / 0.9% is stale by several years.)
- Weekly: `1080 × 0.013 = $14.04`.

## 4. Federal Income Tax Withholding (FIT)

**Federal income tax withholding is OPTIONAL for household employees.** Per Pub 926 (2026):
"you're not required to withhold federal income tax from wages you pay a household employee.
You should withhold federal income tax only if your household employee asks you to withhold it
and you agree." Social Security, Medicare, and CA SDI are *not* optional. *Source: IRS Pub 926
(2026), "Do You Need To Withhold Federal Income Tax?"*

When withheld, use **IRS Pub 15-T (2026)**, Percentage Method. For a post-2020 W-4 with the
Step 2 checkbox unchecked ("Standard withholding"), single filer.

### Weekly percentage-method bracket (Single, Standard) — Pub 15-T (2026), Table 1 (WEEKLY)

| Taxable wage over | but not over | withholding |
|---|---|---|
| $0 | $310 | $0 |
| $310 | $548 | 10% of excess over $310 |
| $548 | $1,279 | $23.80 + 12% of excess over $548 |
| $1,279 | $2,342 | $111.52 + 22% of excess over $1,279 |
| $2,342 | — | $345.38 + 24% of excess over $2,342 |

For a plain post-2020 W-4 (no Step 3/4 entries), the standard deduction is already baked into
the table thresholds, so the "adjusted wage" equals gross pay.

**Worked example — $1,080/week, single:** falls in the $548–$1,279 row →
`23.80 + 0.12 × (1080 − 548) = 23.80 + 63.84 = $87.64/week`.

Cross-check via the **annual** Standard schedule (Pub 15-T 2026, Table 7 ANNUAL, Single):
$0–$16,100 = $0; $16,100–$28,500 = 10%; $28,500–$66,500 = $1,240 + 12%; $66,500–$121,800 =
$5,800 + 22%. Annual wage $56,160 → `1,240 + 0.12 × (56,160 − 28,500) = 4,559.20/yr ÷ 52 =
$87.68/week` — matches the weekly table within rounding. *Source: IRS Pub 15-T (2026),
Percentage Method Tables, Standard Withholding Rate Schedules.*

(The current code's flat 12%-of-gross = $129.60 over-withholds by ~$42/week because it ignores
the standard-deduction $0 bracket.)

## 5. CA State Income Tax Withholding (PIT) — DE 44 Method B

California EDD "California Withholding Schedules for 2026," **Method B – Exact Calculation
Method**. Five steps: (1) low-income exemption test, (2) subtract estimated-deduction
allowances, (3) subtract standard deduction, (4) apply the tax-rate table, (5) subtract the
exemption-allowance credit. *Source: CA EDD 2026 Method B (26methb.pdf).*

Relevant **weekly**, **Single** values (2026):

- **Table 1 — Low Income Exemption (weekly, single):** $363. If gross ≤ $363, withhold $0.
- **Table 3 — Standard Deduction (weekly, single):** $110.
- **Table 4 — Exemption Allowance credit (weekly):** 0 allow = $0.00; **1 allow = $3.24**;
  2 = $6.47; 3 = $9.71; 4 = $12.95 …
- **Table 23 — Tax Rate Table (weekly, Single / Dual-Income Married / Married w/ multiple employers):**

| Taxable income over | but not over | computed tax |
|---|---|---|
| $0 | $213 | 1.100% of excess over $0 |
| $213 | $505 | $2.34 + 2.200% over $213 |
| $505 | $797 | $8.76 + 4.400% over $505 |
| $797 | $1,107 | $21.61 + 6.600% over $797 |
| $1,107 | $1,399 | $42.07 + 8.800% over $1,107 |
| $1,399 | $7,144 | $67.77 + 10.230% over $1,399 |
| $7,144 | $8,573 | $655.48 + 11.330% over $7,144 |
| $8,573 | $14,288 | $817.39 + 12.430% over $8,573 |
| $14,288 | $19,231 | $1,527.76 + 13.530% over $14,288 |
| $19,231 | — | $2,196.55 + 14.630% over $19,231 |

**Worked example — $1,080/week, single, 1 DE 4 allowance:**
1. $1,080 > $363 low-income exemption → withhold.
2. No estimated-deduction allowances → skip.
3. Taxable = `1080 − 110 (std deduction) = $970`.
4. $970 is in $797–$1,107 → `21.61 + 0.066 × (970 − 797) = 21.61 + 11.42 = $33.03`.
5. Subtract 1-allowance credit `$3.24` → **$29.79/week**.

(The current code's flat 6%-of-gross = $64.80 over-withholds by ~$35/week. The spreadsheet's
fixed "$29" estimate matches Method B closely.)

## 6. Employer-side taxes (the rest of the "nanny tax")

The employer owes these on top of net pay; a real payroll service shows total cost of
employment, not just the employee's stub.

- **Employer SS + Medicare:** matches the employee — 6.2% + 1.45% = **7.65%**. *Source: Pub 926 (2026).*
- **FUTA:** 6% of cash wages on the first **$7,000/yr** per employee; a credit of up to 5.4%
  for state unemployment taxes paid on time yields a **net 0.6%** ($42/yr max per employee).
  Applies if you paid $1,000+ in any calendar quarter. Check annually whether CA is a
  credit-reduction state (would raise the net rate); none applied in the source text reviewed.
  *Source: IRS Pub 926 (2026), FUTA section.*
- **CA UI (employer):** new employers pay **3.4%** for the first 2–3 years; experience-rated
  range is **1.5%–6.2%**; taxable wage base **$7,000/yr** per employee. *Source: CA EDD (2026), UI Rate.*
- **CA ETT (employer):** **0.1%**, taxable wage base **$7,000/yr** per employee. *Source: CA EDD (2026), ETT Rate.*

Because UI/ETT/FUTA only apply to the first $7,000 of annual wages, their per-check impact
disappears partway through the year (at $1,080/wk, around week 7).

## 7. Worked pay stub — single CA filer, $1,080/week (2026)

Employee side (what she takes home):

| Line | Rate | Amount |
|---|---|---|
| Gross pay (40 × $27) | | $1,080.00 |
| Social Security (OASDI) | 6.2% | −$66.96 |
| Medicare (HI) | 1.45% | −$15.66 |
| Federal income tax (Pub 15-T, single, plain W-4) | percentage method | −$87.64 |
| CA SDI | 1.3% | −$14.04 |
| CA state income tax (Method B, single, 1 allow) | schedule | −$29.79 |
| **Total employee deductions** | | **−$214.09** |
| **NET PAY** | | **$865.91** |

Employer side (additional cost, early in the year while under the $7,000 UI/ETT/FUTA caps):

| Line | Rate | Amount |
|---|---|---|
| Employer Social Security | 6.2% | $66.96 |
| Employer Medicare | 1.45% | $15.66 |
| FUTA (net) | 0.6% (first $7,000) | $6.48 |
| CA UI (new-employer rate) | 3.4% (first $7,000) | $36.72 |
| CA ETT | 0.1% (first $7,000) | $1.08 |
| **Employer added cost** | | **$126.90** |
| **Total cost of employment (this week)** | | **$1,206.90** |

Compare to the tool's current output for this scenario (**net $793.26**): the tool over-withholds
FIT (flat 12% vs percentage method) and state PIT (flat 6% vs Method B), and uses a stale SDI
rate (0.9% vs 1.3%). Correct 2026 net is **~$865.91**.

## Sources

Primary documents, accessed 2026-07-24 (tax year 2026):

- **IRS Publication 926 (2026), Household Employer's Tax Guide** — https://www.irs.gov/publications/p926
  (SS/Medicare rates & $184,500 wage base, $3,000 coverage threshold, FUTA 6%/0.6%/$7,000/$1,000-quarter,
  optional FIT withholding for household employees).
- **IRS Publication 15-T (2026), Federal Income Tax Withholding Methods** — https://www.irs.gov/publications/p15t
  (percentage-method Standard Withholding Rate Schedules; weekly Table 1 and annual Table 7 used above).
- **CA EDD, Rates and Withholding** — https://edd.ca.gov/en/payroll_taxes/rates_and_withholding/
  (2026 SDI 1.3% & no wage cap; ETT 0.1%/$7,000; UI new-employer 3.4%, range 1.5–6.2%/$7,000).
- **CA EDD, California Withholding Schedules for 2026 — Method B (Exact Calculation)** —
  https://edd.ca.gov/siteassets/files/pdf_pub_ctr/26methb.pdf
  (Table 1 low-income exemption, Table 3 standard deduction, Table 4 exemption-allowance credit,
  Table 23 weekly single tax-rate schedule).

### Not verified / caveats

- **CA credit-reduction status for FUTA in 2026** was not separately confirmed against a DOL
  credit-reduction notice; the 0.6% net rate assumes no reduction. Confirm before relying on it.
- **SS wage base $184,500** is from Pub 926; the SSA COLA fact sheet (ssa.gov/oact/cola/cbb.html)
  blocked automated access, so it is single-sourced (still a primary IRS source).
- **CA UI experience rate** is employer-specific (1.5–6.2%); the example uses the 3.4%
  new-employer rate. A given employer's actual rate comes from their DE 2088.
- Federal W-4 post-2020 has **no allowances**; the "1 W-4 allowance" in the original spreadsheet
  is legacy. FIT above assumes a plain single W-4 (standard withholding).
