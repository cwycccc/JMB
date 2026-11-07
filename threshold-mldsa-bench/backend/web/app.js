'use strict';
// Public historical snapshot generated from experiments/results/results.json.
// It is not a live API result. Statistics below are recomputed from raw samples.
const BUNDLED_EXPERIMENT = {"time_utc":"2026-11-06T14:28:16Z","os":"windows","arch":"amd64","go_version":"go1.24.0","logical_cpus":28,"key_id":"646ea8fa61c605192bf1135c3095f4a62b645d1d3e48e4607d5c2a1938344f80","warmup_excluded":5,"requested":100,"succeeded":100,"samples":[{"index":1,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.8276},{"index":2,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.712},{"index":3,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":51.9588},{"index":4,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.5899},{"index":5,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.2559},{"index":6,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.1792},{"index":7,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":2,"backend_ms":33.5294},{"index":8,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.9208},{"index":9,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.5033},{"index":10,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.8656},{"index":11,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.7515},{"index":12,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":7,"backend_ms":118.9404},{"index":13,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.1016},{"index":14,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.6009},{"index":15,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.6813},{"index":16,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":5,"backend_ms":84.0098},{"index":17,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":3,"backend_ms":52.411},{"index":18,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.1257},{"index":19,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":4,"backend_ms":70.3244},{"index":20,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.2839},{"index":21,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":5,"backend_ms":85.4792},{"index":22,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":3,"backend_ms":51.7651},{"index":23,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":51.2569},{"index":24,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.1804},{"index":25,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":50.6233},{"index":26,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.767},{"index":27,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.2452},{"index":28,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.2213},{"index":29,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":51.7261},{"index":30,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.6061},{"index":31,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":3,"backend_ms":51.6334},{"index":32,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.6019},{"index":33,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":51.4104},{"index":34,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":2,"backend_ms":35.2217},{"index":35,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":9,"backend_ms":152.7055},{"index":36,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.3511},{"index":37,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.8925},{"index":38,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":52.7526},{"index":39,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":52.5998},{"index":40,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.9121},{"index":41,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":3,"backend_ms":52.2272},{"index":42,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":4,"backend_ms":70.2929},{"index":43,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.5895},{"index":44,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.3296},{"index":45,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.8883},{"index":46,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":35.3278},{"index":47,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.0875},{"index":48,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":52.7128},{"index":49,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":35.7083},{"index":50,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.7593},{"index":51,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":2,"backend_ms":35.8917},{"index":52,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":2,"backend_ms":35.244},{"index":53,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.9179},{"index":54,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.8115},{"index":55,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.8565},{"index":56,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.5576},{"index":57,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":5,"backend_ms":86.5782},{"index":58,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":7,"backend_ms":122.8538},{"index":59,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":35.6806},{"index":60,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":4,"backend_ms":70.4809},{"index":61,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.6059},{"index":62,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":2,"backend_ms":36.4186},{"index":63,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":36.255},{"index":64,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.1752},{"index":65,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.0908},{"index":66,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.8118},{"index":67,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.2174},{"index":68,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.6626},{"index":69,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.7062},{"index":70,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":52.5749},{"index":71,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.7677},{"index":72,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.5441},{"index":73,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":35.103},{"index":74,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":19.3055},{"index":75,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.2748},{"index":76,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.1437},{"index":77,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.8592},{"index":78,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.2057},{"index":79,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":35.0556},{"index":80,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.582},{"index":81,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":2,"backend_ms":36.3737},{"index":82,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.404},{"index":83,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.6865},{"index":84,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.7466},{"index":85,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":51.0259},{"index":86,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.1329},{"index":87,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.0739},{"index":88,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.5251},{"index":89,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":51.7788},{"index":90,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.6787},{"index":91,"participants":[0,1,2],"http_status":200,"verified":true,"attempts":2,"backend_ms":36.4202},{"index":92,"participants":[0,1,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":19.2107},{"index":93,"participants":[0,1,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":36.035},{"index":94,"participants":[0,2,3],"http_status":200,"verified":true,"attempts":5,"backend_ms":86.7764},{"index":95,"participants":[0,2,4],"http_status":200,"verified":true,"attempts":2,"backend_ms":34.8616},{"index":96,"participants":[0,3,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":52.601},{"index":97,"participants":[1,2,3],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.1717},{"index":98,"participants":[1,2,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":17.818},{"index":99,"participants":[1,3,4],"http_status":200,"verified":true,"attempts":1,"backend_ms":18.2954},{"index":100,"participants":[2,3,4],"http_status":200,"verified":true,"attempts":3,"backend_ms":53.4554}]};
const BUNDLED_DIGEST = '7644ddab2c79dfacde4da15b11b9db900cf86d6f1f18c6d7b1373099ec7f180a';
const $ = id => document.getElementById(id);
const state = {cluster:null, selected:new Set([0,2,4]), tx:null, record:null, busy:false, operation:'', refreshing:false, samples:[], calls:0, nodes:[], inspected:null, replayTimer:null, replayRound:0, replaying:false};
const fmt = (n,d=2) => Number.isFinite(n) ? n.toFixed(d) : '—';
const bytes = n => Number.isFinite(n) ? n.toLocaleString('zh-CN')+' B' : '—';
function notice(message,error=false) { $('notice').hidden=false; $('notice').textContent=message; $('notice').className='notice'+(error?' error':''); }
function badge(id,text,tone) { $(id).textContent=text; $(id).className='badge '+(tone || (text==='VERIFIED'?'success':text==='FAILED'?'error':text==='SIGNING'?'pending':'neutral')); }
function participants() { return [...state.selected].sort((a,b)=>a-b); }
function parties(ids) { return Array.isArray(ids)&&ids.length?ids.map(id=>'P'+id).join(' / '):'—'; }
function percentile(values,p) { const v=values.slice().sort((a,b)=>a-b); return v.length?v[Math.ceil(v.length*p)-1]:null; }
function parseFen(text) {
  if(!/^(0|[1-9]\d{0,13})(\.\d{1,2})?$/.test(text))throw new Error('金额必须为正数，且最多两位小数。');
  const [yuan,decimal='']=text.split('.'); const fen=BigInt(yuan)*100n+BigInt(decimal.padEnd(2,'0'));
  if(fen<=0n||fen>BigInt(Number.MAX_SAFE_INTEGER))throw new Error('金额超出本演示支持的精确范围。');
  return Number(fen);
}
// Format only saved integer fen. Never turn a floating-point draft into a
// business amount or show an uncreated example as an authenticated instruction.
function formatFen(fen) {
  if(!Number.isSafeInteger(fen)||fen<=0)return '—';
  const amount=BigInt(fen);
  return (amount/100n).toLocaleString('zh-CN')+'.'+(amount%100n).toString().padStart(2,'0');
}
function renderBusiness(r) {
  const tx=r?.transaction||(state.tx?.transaction_id===r?.transaction_id?state.tx:null);
  $('clearing-amount').textContent=tx?formatFen(tx.amount_fen):'—';
  $('payer-bank').textContent=tx?.from_bank||'—';$('payee-bank').textContent=tx?.to_bank||'—';
  $('clearing-context').textContent=tx?'模拟跨行清算申请 · '+tx.currency+' · 指令字段来自已创建交易':'等待创建交易 · 金额由已保存交易读取，不使用表单草稿冒充结果';
}
async function api(path,method='GET',data) {
  const options={method,headers:{Accept:'application/json'},cache:'no-store'};
  if(data!==undefined) {options.headers['Content-Type']='application/json';options.body=JSON.stringify(data);}
  const response=await fetch(path,options); let result;
  try{result=await response.json();}catch{throw new Error('后端返回了非 JSON 响应（HTTP '+response.status+'）');}
  if(!response.ok){const e=new Error(result.error||'请求失败（HTTP '+response.status+'）');e.status=response.status;e.data=result;throw e;}
  return result;
}
function updateControls() {
  const c=state.cluster, signed=state.record?.status==='VERIFIED';
  $('sign').disabled=state.busy||state.replaying||!state.tx||signed||state.record?.status==='SIGNING'||state.selected.size!==3||!c||c.requires_reinitialization;
  $('sign').textContent=state.operation==='sign'?'正在执行网络签名…':'发起分布式签名 →';
  $('create').disabled=state.busy; $('create').textContent=state.operation==='create'?'创建中…':'创建交易 +';
  for(const id of ['tamper','duplicate','shortage']) $(id).disabled=state.busy||!signed||!state.tx;
  $('selection-count').textContent=state.selected.size+' / 3 已选择';
  $('sign-hint').textContent=c?.requires_reinitialization?'Abort 未确认，请整体重新初始化集群。':state.operation==='sign'?'正在等待后端实际结果，不自动重试密码轮次。':state.busy?'正在等待业务 API，不启动新的签名轮次。':signed?'此交易已签名；创建新交易后可继续演示。':!state.tx?'先创建交易，再选择三个参与方。':state.selected.size!==3?'请勾选恰好三个不同节点。':'已准备：'+parties(participants())+' 协同签名。';
  $('replay').disabled=state.busy||state.replaying||!state.record?.verified||!validCounts(state.record.round_message_counts);
  state.nodes.forEach(n=>{n.input.disabled=state.busy;});
}
function validCounts(counts) { return Array.isArray(counts)&&counts.length===3&&counts.every(n=>Number.isSafeInteger(n)&&n>=0); }
function initNodes() {
  for(let id=0;id<5;id++) {
    const cell=document.createElement('div');cell.className='node';
    const button=document.createElement('button');button.type='button';button.className='node-main';button.setAttribute('aria-label','查看 P'+id+' 节点详情');button.setAttribute('aria-haspopup','dialog');
    const icon=document.createElementNS('http://www.w3.org/2000/svg','svg');icon.setAttribute('class','node-bank-icon');icon.setAttribute('viewBox','0 0 24 24');icon.setAttribute('aria-hidden','true');const use=document.createElementNS(icon.namespaceURI,'use');use.setAttribute('href','#icon-bank');icon.append(use);
    const name=document.createElement('strong');name.className='node-name';name.textContent='P'+id;
    const health=document.createElement('span');health.className='node-health';const dot=document.createElement('i');dot.className='node-indicator';const healthText=document.createTextNode('未监测');health.append(dot,healthText);button.append(icon,name,health);
    const label=document.createElement('label');label.className='participant-choice';const input=document.createElement('input');input.type='checkbox';input.setAttribute('aria-label','选择 P'+id+' 参与签名');const choice=document.createElement('span');label.append(input,choice);cell.append(button,label);$('nodes').append(cell);
    button.addEventListener('click',()=>openNode(id));
    input.addEventListener('change',()=>{if(state.busy)return;input.checked?state.selected.add(id):state.selected.delete(id);renderNodes();updateControls();});
    state.nodes.push({cell,button,dot,healthText,input,choice});
  }
  renderNodes();
}
function renderNodes() {
  const r=state.record, sameKey=state.cluster&&r?.key_id===state.cluster.key_id;
  state.nodes.forEach((ui,id)=>{
    const n=state.cluster?.nodes?.find(n=>n.party_id===id), ready=n?.status==='CRYPTO_READY',selected=state.selected.has(id);
    const participated=Boolean(r?.verified&&sameKey&&r.participants?.includes(id));
    ui.cell.classList.toggle('selected',selected);ui.cell.classList.toggle('participated',participated);
    ui.dot.className='node-indicator '+(ready?'online':n?'offline':'');
    ui.healthText.textContent=ready?'密码就绪':n?'不可用':'未监测';ui.input.checked=selected;
    ui.choice.textContent=participated?'已参与':selected?'待参与':'待命';
    ui.button.title=n?n.status+' · 健康检查 '+fmt(n.latency_ms)+' ms':'没有此节点的公开健康数据';
    const wire=document.querySelector('.network-wires [data-party="'+id+'"]');
    wire.classList.toggle('selected',selected);wire.classList.toggle('participated',Boolean(r?.verified&&r.participants?.includes(id)));
  });
  $('network-participants').textContent=r?.verified?'结果参与方：'+parties(r.participants):parties(participants())+' 待参与';
  if(state.inspected!==null)renderNodeDetails(state.inspected);
}
function detailRow(parent,term,value) {
  const div=document.createElement('div'),dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=term;dd.textContent=value;div.append(dt,dd);parent.append(div);
}
function renderNodeDetails(id) {
  const n=state.cluster?.nodes?.find(n=>n.party_id===id),r=state.record;
  $('node-dialog-title').textContent='P'+id+' · 节点公开详情';const list=$('node-details');list.replaceChildren();
  const rows=[['PartyID',String(id)],['节点状态',n?.status||'未获取'],['健康检查延迟',n?fmt(n.latency_ms)+' ms':'未获取'],['集群 KeyID',n?.status==='CRYPTO_READY'?state.cluster.key_id:'身份未经就绪检查确认'],['本次选择',state.selected.has(id)?'已选择':'待命'],['结果参与记录',r?.participants?.includes(id)?'当前交易签名请求的参与方':'当前结果未包含此节点'],['参与轮次',r?.verified&&r.participants?.includes(id)?'本次签名已完成；未采集逐节点轮次事件':'尚无已完成的参与记录'],['公开消息数量','未采集逐节点计数，不使用累计总数平均推算']];
  rows.forEach(([k,v])=>detailRow(list,k,v));
}
function openNode(id) {state.inspected=id;renderNodeDetails(id);$('node-dialog').showModal();}
async function refreshCluster() {
  if(state.refreshing)return;state.refreshing=true;
  try {
    const c=await api('/api/cluster');
    if(!Array.isArray(c.nodes))throw new Error('无效的集群状态');state.cluster=c;
    $('mode').textContent=c.mode==='distributed'?'Distributed':'Local baseline';$('threshold').textContent=c.threshold;$('total').textContent=c.total_parties;
    const ready=c.nodes.filter(n=>n.status==='CRYPTO_READY').length;$('ready').textContent=c.mode==='distributed'?ready:'—';
    $('connection').textContent=c.requires_reinitialization?'需要重新初始化':c.mode==='local'?'已连接 · 本地基线':ready===5?'集群已连接':'已连接 · '+ready+'/5 就绪';
    $('connection-dot').className=ready>=3&&!c.requires_reinitialization?'ready':'error';
    $('cluster-note').textContent=c.mode==='local'?'本地模式，不虚构独立节点状态':c.requires_reinitialization?'已停止后续签名':'公开健康检查 · '+new Date().toLocaleTimeString('zh-CN',{hour12:false});
    if(!state.record)$('key-id').textContent=c.key_id;renderNodes();renderVerification();updateControls();
  } catch {
    state.cluster=null;$('connection').textContent='后端连接失败';$('connection-dot').className='error';$('ready').textContent='—';$('cluster-note').textContent='健康状态未知，不沿用旧状态';renderNodes();renderVerification();updateControls();
  } finally {state.refreshing=false;}
}
function check(id,pass) { $(id).textContent=pass===true?'✓':pass===false?'!':'—';$(id).className=pass===true?'pass':''; }
function renderVerification() {
  const r=state.record;check('key-check',r?.verified?(state.cluster?r.key_id===state.cluster.key_id:null):null);
  check('signature-check',r?.verified===true?true:r?.status==='FAILED'?false:null);
}
function renderRecord(r) {
  stopReplay(false);state.record=r;badge('transaction-status',r.status);badge('result-status',r.status);badge('protocol-state',r.status==='PENDING'?'等待签名':r.status);
  renderBusiness(r);
  $('current-id').textContent=r.transaction_id||'未创建';$('key-id').textContent=r.key_id||state.cluster?.key_id||'—';$('session-id').textContent=r.session_id||'尚未创建';
  const measured=Number.isFinite(r.duration_ms)&&r.duration_ms>0;
  $('latest-duration').textContent=measured?fmt(r.duration_ms,2):'—';$('attempts').textContent=r.attempts>0?r.attempts:'—';
  $('signing-time').textContent=measured?fmt(r.signing_duration_ms,4)+' ms':'—';$('verification-time').textContent=measured?fmt(r.verification_duration_ms,4)+' ms':'—';
  $('result-participants').textContent=parties(r.participants);
  const card=document.querySelector('.outcome-card');card.classList.toggle('verified',r.verified===true);card.classList.toggle('failed',r.status==='FAILED');
  $('outcome-title').textContent=r.verified?'VERIFIED':r.status==='FAILED'?'FAILED':r.status==='SIGNING'?'SIGNING':'等待签名';
  $('verify-symbol').textContent=r.verified?'✓':'◇';$('verify-text').textContent=r.verified?'后端公钥独立验签通过':r.status==='FAILED'?'请求未成功；查看错误与安全停止信息':'创建交易后，由后端驱动真实三轮协议';
  const counts=validCounts(r.round_message_counts)?r.round_message_counts:null,traffic=r.traffic?.rounds;
  for(let i=1;i<=3;i++) {
    const t=traffic?.[i-1];$('round'+i+'-count').textContent=counts?counts[i-1]+' 条':'—';$('mini-count'+i).textContent=counts?counts[i-1]+' 条消息':'—';
    $('round'+i+'-sent').textContent=bytes(t?.http_sent_bytes);$('round'+i+'-received').textContent=bytes(t?.http_received_bytes);$('round'+i+'-payload').textContent=bytes(t?.protocol_response_bytes);
  }
  $('http-bytes').textContent=traffic?bytes(traffic.reduce((a,x)=>a+x.http_sent_bytes+x.http_received_bytes,0)):'未采集';
  $('protocol-bytes').textContent=traffic?bytes(traffic.reduce((a,x)=>a+x.protocol_response_bytes,0)):'未采集';
  $('attempt-summary').textContent=r.attempts>0?'累计 '+r.attempts+' 次 · 逐次事件未提供':'尚无逐次事件';
  $('history-session-label').textContent=r.session_id?'Session '+r.session_id:'尚未创建签名会话';$('history-outcome').textContent=r.status;
  $('history-observation').textContent=counts?'观测到 Round 1 / 2 / 3 累计 '+counts.join(' / ')+' 条响应。未按 Attempt 拆分。':r.status==='FAILED'?'当前失败响应没有完整的轮次计数；不将未知次数显示为零。':'尚无三轮公开消息统计。';
  const extra=r.abort_confirmed===false?'本次响应明确表示 Abort 未确认，必须重新初始化。':r.abort_confirmed===true?'本次失败响应明确表示 Abort 已确认。':'没有逐 Attempt 的 Abort 确认明细。';
  $('attempt-boundary').textContent='当前 API 未提供逐 Attempt 事件和 Combine 拒绝原因。不从累计次数生成 Attempt 0/1/2 或推断拒绝原因。'+extra;
  $('signature').value=r.signature_base64||'';$('copy-signature').disabled=!r.signature_base64;
  try{$('signature-size').textContent=r.signature_base64?atob(r.signature_base64).length.toLocaleString('zh-CN')+' bytes（Base64 展开可见）':'—';}catch{$('signature-size').textContent='编码无法解析';}
  renderNodes();renderVerification();updateControls();
}
function svgElement(svg,name,attrs,text) {
  const el=document.createElementNS('http://www.w3.org/2000/svg',name);
  for(const [k,v] of Object.entries(attrs))el.setAttribute(k,String(v));if(text!==undefined)el.textContent=text;svg.append(el);return el;
}
function lineChart(id,values,height=170) {
  const svg=$(id);svg.replaceChildren();svg.setAttribute('viewBox','0 0 560 '+height);
  const left=38,right=543,top=20,bottom=height-29,limit=Math.max(25,Math.ceil(Math.max(0,...values.map(v=>v.value))/25)*25);
  for(let j=0;j<=3;j++) {
    const y=bottom-j*(bottom-top)/3;svgElement(svg,'line',{x1:left,x2:right,y1:y,y2:y,stroke:'#dfe7db','stroke-dasharray':'3 5'});
    svgElement(svg,'text',{x:left-7,y:y+4,'text-anchor':'end',fill:'#687d90','font-size':11},fmt(limit*j/3,0));
  }
  svgElement(svg,'text',{x:3,y:12,fill:'#687d90','font-size':11},'ms');if(!values.length)return;
  const first=values[0].index,last=values.at(-1).index;
  const points=values.map(v=>[values.length===1?(left+right)/2:left+(v.index-first)*(right-left)/Math.max(last-first,1),bottom-v.value/limit*(bottom-top)]);
  if(points.length>1) {
    svgElement(svg,'polygon',{points:points[0][0]+','+bottom+' '+points.map(p=>p.join(',')).join(' ')+' '+points.at(-1)[0]+','+bottom,fill:'#eaf2f9'});
    svgElement(svg,'polyline',{points:points.map(p=>p.join(',')).join(' '),fill:'none',stroke:'#2c6c97','stroke-width':1.8,'stroke-linejoin':'round'});
  }
  points.forEach(([cx,cy],i)=>{const circle=svgElement(svg,'circle',{cx,cy,r:values.length>40?2:3,fill:'#1b628f'});const title=document.createElementNS(svg.namespaceURI,'title');title.textContent='样本 '+values[i].index+' · '+fmt(values[i].value,4)+' ms';circle.append(title);});
  svgElement(svg,'text',{x:left,y:height-7,fill:'#687d90','font-size':11},'样本 '+first);
  if(values.length>1)svgElement(svg,'text',{x:right,y:height-7,'text-anchor':'end',fill:'#687d90','font-size':11},String(last));
}
function renderPerformance() {
  $('sample-count').textContent=state.calls+' 次请求';$('success-rate').textContent=state.calls?fmt(state.samples.length/state.calls*100,0)+'%':'—';
  const values=state.samples.map(s=>s.value);$('mean').textContent=values.length?fmt(values.reduce((a,b)=>a+b,0)/values.length)+' ms':'—';
  $('p95').textContent=values.length>=30?fmt(percentile(values,.95))+' ms':'样本不足';$('session-p50').textContent=values.length>=30?fmt(percentile(values,.5))+' ms':'样本不足';
  $('session-range').textContent=values.length?values.length+' 个成功样本':'暂无样本';$('chart-empty').hidden=values.length>0;
  lineChart('latency-chart',state.samples.slice(-40));
}
function experimentData(data) {
  if(!data||!Array.isArray(data.samples)||data.samples.length<30||data.samples.length>1000||data.requested!==data.samples.length||!/^[a-f0-9]{64}$/.test(data.key_id||'')||!Number.isFinite(Date.parse(data.time_utc)))throw new Error('需要包含 30–1000 条原始样本、KeyID、时间和一致样本数的正式实验 JSON。');
  const seen=new Set(),samples=data.samples.map(s=>{
    if(!Number.isSafeInteger(s.index)||s.index<1||seen.has(s.index)||typeof s.verified!=='boolean'||!Number.isFinite(s.backend_ms)||s.backend_ms<0||s.backend_ms>3600000||!Number.isInteger(s.http_status)||s.http_status<100||s.http_status>599||s.verified&&(s.http_status!==200||s.backend_ms<=0))throw new Error('实验包含重复编号、无效状态或无效耗时。');
    seen.add(s.index);return {index:s.index,verified:s.verified,http_status:s.http_status,backend_ms:s.backend_ms};
  }).sort((a,b)=>a.index-b.index);
  const success=samples.filter(s=>s.verified),v=success.map(s=>s.backend_ms);
  if(success.length!==data.succeeded||success.length<30)throw new Error('成功数与原始样本不一致，或成功样本不足 30 条。');
  return {samples,success,mean:v.reduce((a,b)=>a+b,0)/v.length,p50:percentile(v,.5),p95:percentile(v,.95),keyID:data.key_id,time:data.time_utc,warmup:Number.isSafeInteger(data.warmup_excluded)&&data.warmup_excluded>=0?data.warmup_excluded:null,environment:[data.os,data.arch,data.go_version].map(v=>typeof v==='string'?v.slice(0,60):'未知').join(' / ')};
}
function formalBars(d) {
  const svg=$('formal-bars');svg.replaceChildren();const limit=Math.max(25,Math.ceil(Math.max(d.mean,d.p50,d.p95)/25)*25),left=62,width=394;
  for(let j=0;j<=4;j++){const x=left+j*width/4;svgElement(svg,'line',{x1:x,x2:x,y1:16,y2:161,stroke:'#e2e8dd','stroke-dasharray':'3 5'});svgElement(svg,'text',{x,y:187,'text-anchor':'middle',fill:'#687d90','font-size':12},fmt(limit*j/4,0));}
  [['平均',d.mean,'#1c5783'],['P50',d.p50,'#7a9db6'],['P95',d.p95,'#c5a36a']].forEach(([label,value,fill],i)=>{const y=24+i*48;svgElement(svg,'text',{x:0,y:y+19,fill:'#526b57','font-size':12},label);svgElement(svg,'rect',{x:left,y,width:value/limit*width,height:28,rx:3,fill});svgElement(svg,'text',{x:Math.min(506,left+value/limit*width+8),y:y+19,fill:'#304f69','font-size':12},fmt(value));});
}
function renderFormal(raw,source,digest) {
  const d=experimentData(raw);
  $('formal-kpi-mean').textContent=fmt(d.mean);$('formal-kpi-caption').textContent='历史 '+d.samples.length+' 次实验 · 非本次签名耗时';
  $('formal-rate').textContent=fmt(d.success.length/d.samples.length*100,0)+'%';$('formal-ratio').textContent='历史正式实验 '+d.success.length+' / '+d.samples.length+' · 非实时';
  $('formal-mean').textContent=fmt(d.mean)+' ms';$('formal-p50').textContent=fmt(d.p50)+' ms';$('formal-p95').textContent=fmt(d.p95)+' ms';$('formal-count').textContent=d.success.length+' / '+d.samples.length;
  $('formal-warmup').textContent=d.warmup===null?'预热数量未报告':'另有 '+d.warmup+' 次预热，不计入统计';
  const date=new Date(d.time).toLocaleString('zh-CN',{timeZone:'Asia/Shanghai',hour12:false});
  const knownArchive=digest===BUNDLED_DIGEST;
  $('formal-description').textContent='历史实验 · '+date+'（北京时间）· '+(knownArchive?'串行本机回环网络。':'导入文件的负载与网络条件以原始实验说明为准。')+'数值由原始样本重新计算，与当前集群实例分开；浏览器不重新执行历史验签。';
  $('formal-badge').textContent=d.samples.length+' 条历史记录';$('formal-series-count').textContent=d.success.length+' 条原始成功测量值';
  $('formal-source').textContent=source;$('formal-time').textContent=date+'（Asia/Shanghai）';$('formal-key').textContent=d.keyID;
  $('formal-environment').textContent=d.environment+(knownArchive?' · localhost HTTP/1.1':' · 网络条件未在此 JSON 中声明');$('formal-digest').textContent=digest||'未计算';formalBars(d);
  lineChart('formal-series',d.success.map(s=>({index:s.index,value:s.backend_ms})),200);
}
function replayStep(round) {
  for(let i=1;i<=3;i++){$('round'+i).classList.toggle('active',i===round);$('mini-round'+i).classList.toggle('active',i===round);}
  const r=state.record;
  $('replay-status').textContent='历史汇总回放 · Round '+round+' / 3 · 原记录累计 '+r.round_message_counts[round-1]+' 条响应。此回放不还原逐 Attempt 事件或真实发送时刻，不发起网络请求。';
}
function stopReplay(announce=true) {
  if(state.replayTimer)clearInterval(state.replayTimer);state.replayTimer=null;
  const was=state.replaying;state.replaying=false;$('topology').classList.remove('replaying');$('replay-stop').hidden=true;
  for(let i=1;i<=3;i++){$('round'+i).classList.remove('active');$('mini-round'+i).classList.remove('active');}
  if(was&&announce)$('replay-status').textContent='历史汇总回放已结束。仅使用已返回的累计轮次统计，未触发签名请求。';
  if(!announce)$('replay-status').hidden=true;updateControls();
}
function startReplay() {
  if(!state.record?.verified||!validCounts(state.record.round_message_counts)||state.busy)return;
  stopReplay(false);state.replaying=true;state.replayRound=1;$('topology').classList.add('replaying');$('replay-status').hidden=false;$('replay-stop').hidden=false;replayStep(1);updateControls();
  state.replayTimer=setInterval(()=>{state.replayRound++;if(state.replayRound>3){stopReplay();return;}replayStep(state.replayRound);},1700);
}
function resetSecurity() {
  for(const kind of ['tamper','duplicate','shortage']){badge(kind+'-status','待执行','neutral');$(kind+'-feedback').hidden=true;$(kind+'-feedback').replaceChildren();}
  check('integrity-check',null);$('security-result').textContent='等待当前交易签名完成后执行真实测试用例。';
}
async function createTransaction(event) {
  event.preventDefault();if(state.busy)return;stopReplay(false);
  try {
    const tx={version:'clearing-v1',transaction_id:$('transaction-id').value.trim(),from_bank:$('from-bank').value.trim(),to_bank:$('to-bank').value.trim(),amount_fen:parseFen($('amount').value.trim()),currency:'CNY',nonce:crypto.randomUUID()};
    state.busy=true;state.operation='create';updateControls();await api('/api/transactions','POST',tx);state.tx=tx;
    renderRecord({transaction_id:tx.transaction_id,status:'PENDING',participants:[]});resetSecurity();
    const r=await api('/api/transactions/'+encodeURIComponent(tx.transaction_id));renderRecord(r);
    notice('交易 '+tx.transaction_id+' 已创建：'+tx.amount_fen.toLocaleString('zh-CN')+' 分。请选择三个节点并发起签名。');
  }catch(e){notice(e.message,true);}finally{state.busy=false;state.operation='';updateControls();}
}
async function signCurrent() {
  if(state.busy||!state.tx)return;stopReplay(false);state.busy=true;state.operation='sign';updateControls();
  badge('protocol-state','SIGNING');badge('transaction-status','SIGNING');$('outcome-title').textContent='SIGNING';$('verify-text').textContent='正在等待后端实际结果，不模拟实时轮次';notice('后端正在驱动真实三轮协议；密码轮次 HTTP 请求不会自动重发。');state.calls++;
  const requestIndex=state.calls;
  try {
    const r=await api('/api/transactions/'+encodeURIComponent(state.tx.transaction_id)+'/sign','POST',{participants:participants()});
    renderRecord(r);if(r.verified&&Number.isFinite(r.duration_ms))state.samples.push({index:requestIndex,value:r.duration_ms});
    notice(r.verified?r.transaction_id+' 已独立验签通过；'+r.attempts+' 次尝试，总耗时 '+fmt(r.duration_ms,4)+' ms。':'未确认验签成功。',!r.verified);
  }catch(e) {
    if(e.data?.transaction_id)renderRecord(e.data);
    else{try{renderRecord(await api('/api/transactions/'+encodeURIComponent(state.tx.transaction_id)));}catch{$('outcome-title').textContent='状态未确认';badge('protocol-state','状态未确认');}}
    notice(e.message+'。检查交易状态；未确认前不要重新发起签名。',true);
  }finally{state.busy=false;state.operation='';renderPerformance();updateControls();refreshCluster();}
}
function testFeedback(kind,passed,rows,request) {
  badge(kind+'-status',passed?'测试通过':'未通过',passed?'success':'error');const el=$(kind+'-feedback');el.replaceChildren();el.hidden=false;
  const title=document.createElement('h4');title.textContent=passed?'实际结果符合预期':'实际结果不符合预期';title.className=passed?'pass':'fail';el.append(title);
  const dl=document.createElement('dl');rows.forEach(([k,v])=>detailRow(dl,k,String(v)));el.append(dl);
  const details=document.createElement('details'),summary=document.createElement('summary'),pre=document.createElement('pre');summary.textContent='查看实际请求';pre.className='request-body';pre.textContent=JSON.stringify(request,null,2);details.append(summary,pre);el.append(details);
}
async function securityTest(kind) {
  if(!state.tx||state.busy||!state.record?.verified)return;stopReplay(false);state.busy=true;state.operation='security';updateControls();badge(kind+'-status','执行中','pending');
  const path='/api/transactions/'+encodeURIComponent(state.tx.transaction_id);let request;
  try {
    if(kind==='tamper') {
      if(state.tx.amount_fen>=Number.MAX_SAFE_INTEGER)throw new Error('金额达到精确上限，无法增加 1 分。');
      const original=await api(path+'/verify','POST',state.tx),modified={...state.tx,amount_fen:state.tx.amount_fen+1};request={method:'POST',path:path+'/verify',original_amount_fen:state.tx.amount_fen,candidate:modified};
      const changed=await api(path+'/verify','POST',modified),fetched=await api(path);
      const unchanged=JSON.stringify(fetched.transaction)===JSON.stringify(state.tx)&&fetched.signature_base64===state.record.signature_base64;
      const passed=original.verified===true&&changed.verified===false&&unchanged;check('integrity-check',unchanged);
      testFeedback(kind,passed,[['HTTP 响应','200 / 200'],['原交易验签',original.verified],['金额 +1 分验签',changed.verified],['已保存交易',unchanged?'GET 确认未变化':'内容或签名发生变化']],request);
      $('security-result').textContent='金额篡改用例：原文 '+original.verified+'，副本 '+changed.verified+'；已保存交易'+(unchanged?'未变化。':'发生异常。');
    }else {
      const originalParticipants=state.record.participants.slice();
      const selected=kind==='shortage'?originalParticipants.slice(0,2):originalParticipants;request={method:'POST',path:path+'/sign',body:{participants:selected}};
      const expected=kind==='shortage'?400:409;let actual=200,detail='请求意外被接受';
      try{await api(request.path,'POST',request.body);}catch(e){if(!e.status)throw e;actual=e.status;detail=e.message;}
      const fetched=await api(path),unchanged=fetched.session_id===state.record.session_id&&fetched.signature_base64===state.record.signature_base64&&fetched.status==='VERIFIED';
      const passed=actual===expected&&unchanged;
      testFeedback(kind,passed,[['选择参与方',parties(selected)+'（'+selected.length+' 个）'],['签名门限','3 / 5'],['预期 HTTP',expected],['实际 HTTP',actual],['原会话和签名',unchanged?'GET 确认未变化':'发生变化或未确认'],['后端响应',detail]],request);
      $('security-result').textContent=(kind==='shortage'?'不足门限的请求':'重复签名请求')+'：预期 HTTP '+expected+'，实际 HTTP '+actual+'。'+(unchanged?'已确认原会话与签名未改变。':'结果未确认。')+(kind==='shortage'?'此为入参校验，不是仅两个节点在线的 HTTP 503 故障测试。':'');
    }
  }catch(e){testFeedback(kind,false,[['执行错误',e.message]],request||{operation:kind});$('security-result').textContent='用例未完成：'+e.message;}
  finally{state.busy=false;state.operation='';updateControls();}
}
async function loadExperiment(event) {
  const file=event.target.files?.[0];if(!file)return;
  try{
    if(file.size>4*1024*1024)throw new Error('实验 JSON 不得超过 4 MiB。');
    const text=await file.text(),raw=JSON.parse(text),hash=await crypto.subtle.digest('SHA-256',new TextEncoder().encode(text));
    const digest=Array.from(new Uint8Array(hash),b=>b.toString(16).padStart(2,'0')).join('');
    renderFormal(raw,'浏览器本地加载：'+file.name,digest);
    $('experiment-feedback').textContent='已从本地 JSON 重新计算历史统计。文件未上传；当前会话数据没有改变。';
  }catch(e){$('experiment-feedback').textContent='加载失败，保留上一份有效实验：'+e.message;}
  finally{event.target.value='';}
}
function boot() {
  initNodes();renderPerformance();updateControls();
  try{renderFormal(BUNDLED_EXPERIMENT,'内置历史快照 · experiments/results/results.json',BUNDLED_DIGEST);}catch(e){$('experiment-feedback').textContent='历史实验不可用：'+e.message;}
  $('transaction-id').value='TX'+new Date().toISOString().replace(/\D/g,'').slice(0,14);
  $('transaction-form').addEventListener('submit',createTransaction);$('sign').addEventListener('click',signCurrent);
  for(const kind of ['tamper','duplicate','shortage'])$(kind).addEventListener('click',()=>securityTest(kind));
  $('replay').addEventListener('click',startReplay);$('replay-stop').addEventListener('click',()=>stopReplay());
  $('node-close').addEventListener('click',()=>$('node-dialog').close());$('node-dialog').addEventListener('close',()=>{state.inspected=null;});
  $('experiment-file').addEventListener('change',loadExperiment);
  $('copy-signature').addEventListener('click',async()=>{try{await navigator.clipboard.writeText($('signature').value);notice('已复制公开的 Base64 签名，不含秘密份额。');}catch{notice('剪贴板不可用，请展开签名手动复制。',true);}});
  $('refresh').addEventListener('click',async()=>{await refreshCluster();if(state.tx&&!state.busy){try{renderRecord(await api('/api/transactions/'+encodeURIComponent(state.tx.transaction_id)));}catch(e){notice(e.message,true);}}});
  refreshCluster();setInterval(refreshCluster,5000);
}
if(typeof document!=='undefined')boot();

