// Run with Node.js: node scripts/test-history-bins.cjs
// Exercise the exact pure display helpers embedded in the local dashboard.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const source = fs.readFileSync(path.join(__dirname, '../dashboard.go'), 'utf8');
const start = source.indexOf('function historyInterval(');
const end = source.indexOf('function attachHistoryDetails(', start);
assert.ok(start >= 0 && end > start);
const context = vm.createContext({});
vm.runInContext(source.slice(start, end), context);
const { historyInterval, historyBins, historyRange, historyZoomRange, forecastResetLead } = context;
const minute = 60000, hour = 60 * minute, day = 24 * hour;
for (const [span, expected] of [[0, minute], [5 * hour, 10 * minute], [24 * hour, 30 * minute], [7 * day, 4 * hour], [30 * day, day]]) {
 assert.equal(historyInterval(span), expected);
}
assert.ok(30 * day / historyInterval(30 * day, 2) <= 3, 'long-range mobile tick labels must stay sparse');
const rangeEnd = Date.UTC(2026, 9, 15, 15, 30);
for (const [days, interval] of [[1, 30 * minute], [7, 4 * hour], [30, day]]) {
 const range = historyRange(days, rangeEnd);
 assert.equal(range.end, rangeEnd);
 assert.equal(range.start, rangeEnd - days * day);
 assert.equal(range.interval, interval);
}
const at = Date.UTC(2026, 9, 15, 10);
const sample = (offset, left, reset = 100, window = 3600) => Object.freeze({at: new Date(at + offset).toISOString(), left, reset_at: reset, window_seconds: window, source: 'live'});
const input = Object.freeze([sample(0, 80), sample(2 * minute, 75), sample(9 * minute, 0), sample(10 * minute, 62), sample(19 * minute, 60)]);
const before = JSON.stringify(input);
let bins = historyBins(input, 10 * minute);
assert.deepEqual(Array.from(bins, b => b.s.left), [0, 60]);
assert.equal(bins[0].s, input[2]);
assert.equal(bins[1].s, input[4]);
assert.equal(JSON.stringify(input), before);
assert.equal(bins[1].bucket - bins[0].bucket, 1);
// Resets A -> B -> A within one bucket must keep each contiguous cycle.
bins = historyBins([sample(0, 50), sample(minute, 48), sample(2 * minute, 100, 200), sample(3 * minute, 99, 200), sample(4 * minute, 90)], 10 * minute);
assert.deepEqual(Array.from(bins, b => [b.s.left, b.segment]), [[48, 0], [99, 1], [90, 2]]);
bins = historyBins([sample(0, 70), sample(minute, 60, 100, 7200)], 10 * minute);
assert.equal(bins.length, 2);
assert.notEqual(bins[0].segment, bins[1].segment);
// Empty buckets remain empty; rendering can break the line at their gap.
bins = historyBins([sample(0, 80), sample(40 * minute, 50)], 10 * minute);
assert.equal(bins.length, 2);
assert.equal(bins[1].bucket - bins[0].bucket, 4);
assert.equal(historyBins([], minute).length, 0);
assert.equal(historyBins([input[0]], minute)[0].s, input[0]);
// Timestamp ties retain the last supplied observation; bucket placement is stable.
assert.equal(historyBins([sample(0, 80), sample(0, 70)], minute)[0].s.left, 70);
assert.equal(historyBins(input.slice(1), 10 * minute)[0].bucket, historyBins(input, 10 * minute)[0].bucket);
const full = historyRange(7, at + 6 * hour);
const zoom = historyZoomRange([sample(0, 80), sample(5 * hour, 40)], full);
assert.ok(zoom.start >= full.start && zoom.end <= full.end);
assert.ok(zoom.start < at && zoom.end > at + 5 * hour);
assert.ok(zoom.end - zoom.start < 6 * hour);
assert.equal(zoom.interval, 10 * minute);
assert.equal(historyZoomRange([], full), full);
assert.equal(historyZoomRange([sample(0, 80)], full), full);
assert.equal(historyZoomRange([sample(0, 80), sample(0, 60)], full), full);
const complete = historyZoomRange([{at:new Date(full.start).toISOString()},{at:new Date(full.end).toISOString()}], full);
assert.equal(complete.start, full.start);
assert.equal(complete.end, full.end);
assert.ok(historyZoomRange([sample(0, 80), sample(1, 40)], historyRange(1, at + 1)).end > at);
const prediction = {estimated_exhaustion:new Date(at).toISOString(),exhausts_before_reset:true};
for(const [gap,label] of [[5*day+6*hour,'5일 6시간'],[day,'1일'],[2*hour+15*minute,'2시간 15분'],[45*minute,'45분'],[30000,'1분 미만']]) {
 assert.equal(forecastResetLead({...prediction,reset_at:(at+gap)/1000}), '초기화보다 약 '+label+' 먼저 소진 예상');
}
for(const invalid of [{}, {reset_at:0}, {reset_at:at/1000}, {reset_at:(at-minute)/1000}, {reset_at:(at+day)/1000,exhausts_before_reset:false}, {reset_at:(at+day)/1000,estimated_exhaustion:'invalid'}]) {
 assert.equal(forecastResetLead({...prediction,...invalid}), '');
}
console.log('PASS: interval selection, last values incl zero, immutable input, reset/window boundaries, sparse/empty/single/tied records, stable bucket alignment');
