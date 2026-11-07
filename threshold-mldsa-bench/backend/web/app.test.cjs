// Pure frontend checks; no browser, network, build tool or npm dependency.
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const crypto = require('node:crypto');
const source = fs.readFileSync(path.join(__dirname, 'app.js'), 'utf8');
const context = vm.createContext({});
vm.runInContext(source, context);
const run = code => vm.runInContext(code, context);

test('decimal currency is converted exactly without floating point multiplication', () => {
  assert.equal(run("parseFen('1000000.00')"), 100000000);
  assert.equal(run("parseFen('0.01')"), 1);
  assert.equal(run("parseFen('123.4')"), 12340);
  for (const amount of ['0', '-1', '1.001', 'NaN', '1e4', '90071992547409.92']) {
    assert.throws(() => run('parseFen('+JSON.stringify(amount)+')'));
  }
});

test('clearing hero formats saved integer fen exactly and never invents an amount',()=>{
  assert.equal(run('formatFen(100000000)'), '1,000,000.00');
  assert.equal(run('formatFen(1)'), '0.01');
  assert.equal(run('formatFen(12340)'), '123.40');
  assert.equal(run('formatFen(Number.MAX_SAFE_INTEGER)'), '90,071,992,547,409.91');
  for(const value of ['0','-1','1.2','NaN','undefined'])assert.equal(run('formatFen('+value+')'),'—');
});

test('business hero uses the saved transaction rather than a draft or prototype example',()=>{
  const sandbox=vm.createContext({});
  vm.runInContext(source,sandbox);
  const fields={};
  sandbox.document={getElementById:id=>fields[id]||(fields[id]={textContent:''})};
  vm.runInContext("renderBusiness({status:'PENDING'})",sandbox);
  assert.equal(fields['clearing-amount'].textContent,'—');
  assert.equal(fields['payer-bank'].textContent,'—');
  vm.runInContext("state.tx={transaction_id:'REAL',amount_fen:12345,from_bank:'机构甲',to_bank:'机构乙',currency:'CNY'};renderBusiness({transaction_id:'REAL',status:'PENDING'})",sandbox);
  assert.equal(fields['clearing-amount'].textContent,'123.45');
  assert.equal(fields['payer-bank'].textContent,'机构甲');
  vm.runInContext("renderBusiness({transaction_id:'OTHER',status:'PENDING'})",sandbox);
  assert.equal(fields['clearing-amount'].textContent,'—');
});

test('bundled archive is derived from the actual public experiment and all 100 raw samples', () => {
  const file=fs.readFileSync(path.join(__dirname,'../../experiments/results/results.json'));
  const archive=JSON.parse(file);
  assert.equal(run('BUNDLED_DIGEST'), crypto.createHash('sha256').update(file).digest('hex'));
  assert.equal(run('BUNDLED_EXPERIMENT.key_id'), archive.key_id);
  assert.equal(run('BUNDLED_EXPERIMENT.time_utc'), archive.time_utc);
  assert.equal(run('BUNDLED_EXPERIMENT.samples.length'), archive.samples.length);
  assert.deepEqual(JSON.parse(run('JSON.stringify(BUNDLED_EXPERIMENT.samples.map(s=>s.backend_ms))')), archive.samples.map(s=>s.backend_ms));
  const stats=JSON.parse(run('JSON.stringify(experimentData(BUNDLED_EXPERIMENT))'));
  assert.equal(stats.success.length,100);
  assert.ok(Math.abs(stats.mean-archive.backend_ms_successes.Mean)<1e-9);
  assert.equal(stats.p50,archive.backend_ms_successes.P50);
  assert.equal(stats.p95,archive.backend_ms_successes.P95);
});

test('nearest-rank quantiles and strict experiment import validation', () => {
  assert.equal(run('percentile([4,1,3,2],.5)'),2);
  assert.equal(run('percentile([4,1,3,2],.95)'),4);
  for (const mutate of [
    'x.succeeded--', 'x.requested--', 'x.key_id="wrong"', 'x.samples[0].backend_ms=-1',
    'x.samples[0].backend_ms=Infinity', 'x.samples[0].index=x.samples[1].index',
    'x.samples[0].http_status=503', 'x.samples=x.samples.slice(0,1); x.requested=1; x.succeeded=1'
  ]) assert.throws(()=>run('(function(){const x=JSON.parse(JSON.stringify(BUNDLED_EXPERIMENT));'+mutate+'; return experimentData(x);})()'));
});

test('all referenced frontend IDs exist, with no duplicate DOM IDs', () => {
  const html=fs.readFileSync(path.join(__dirname,'index.html'),'utf8');
  const ids=[...html.matchAll(/\bid="([^"]+)"/g)].map(m=>m[1]);
  assert.equal(ids.length,new Set(ids).size);
  for(const m of source.matchAll(/\$\('([^']+)'\)/g))assert.ok(ids.includes(m[1]),'missing '+m[1]);
  assert.ok(!/fetch\([^)]*\/internal\//.test(source));
  assert.ok(source.includes('values.length>=30'));
});
