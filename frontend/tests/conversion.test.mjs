import { test } from 'node:test'
import assert from 'node:assert/strict'
import { parseAmount, localDate } from './.compiled/conversion.js'

for (const [text, expected] of [
  ['100,25', 100.25], ['100.25', 100.25], [' 12,5 ', 12.5], ['0', 0],
  ['', null], ['   ', null], ['0x10', null], ['0b10', null], ['1e2', null],
  ['-1', null], ['NaN', null], ['Infinity', null], ['1e309', null], ['1,2,3', null],
  ['1 234,5', null], ['1.234,5', null], ['12,', null], ['.5', null],
]) {
  test(`amount ${JSON.stringify(text)} -> ${expected}`, () => assert.equal(parseAmount(text), expected))
}

test('calendar date uses local fields, including at midnight', () => {
  assert.equal(localDate(new Date(2026, 3, 14, 0, 15)), '2026-04-14')
})
