import Decimal from 'decimal.js';

// Global configuration for financial calculations (2 decimal places precision)
Decimal.set({ precision: 20, rounding: Decimal.ROUND_HALF_UP });

export { Decimal };

/**
 * Creates a safe Decimal object from string, number, or Decimal instance.
 */
export function toDecimal(val: string | number | Decimal): Decimal {
  if (val instanceof Decimal) return val;
  try {
    return new Decimal(val || 0);
  } catch {
    return new Decimal(0);
  }
}

/**
 * Safely parses user input string containing comma or dot (e.g. "150,50" or "150.50") into Decimal.
 * Returns null if invalid or NaN.
 */
export function parseDecimalAmount(val: string): Decimal | null {
  if (!val || typeof val !== 'string') return null;
  const normalized = val.trim().replace(',', '.');
  if (normalized === '' || isNaN(Number(normalized))) return null;
  try {
    const d = new Decimal(normalized);
    return d.isNaN() ? null : d;
  } catch {
    return null;
  }
}

/**
 * Formats a Decimal, number, or string as Turkish Lira currency (e.g. ₺15.000,50 or ₺-3.200,00).
 */
export function formatTL(val: string | number | Decimal): string {
  const d = toDecimal(val);
  const num = d.toNumber();

  return new Intl.NumberFormat('tr-TR', {
    style: 'currency',
    currency: 'TRY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num);
}

/**
 * Adds two monetary values safely using decimal.js.
 */
export function addDecimal(a: string | number | Decimal, b: string | number | Decimal): Decimal {
  return toDecimal(a).add(toDecimal(b));
}

/**
 * Subtracts b from a safely using decimal.js.
 */
export function subDecimal(a: string | number | Decimal, b: string | number | Decimal): Decimal {
  return toDecimal(a).sub(toDecimal(b));
}
