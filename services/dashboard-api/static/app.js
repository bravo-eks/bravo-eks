const API_BASE='';
const $=s=>document.querySelector(s),$$=s=>Array.from(document.querySelectorAll(s));let state={orders:[],products:[],shipments:[],ledger:[],source:'loading'};
const money=(n,c='GBP')=>new Intl.NumberFormat('en-GB',{style:'currency',currency:c}).format(Number(n||0));const label=s=>(s||'').replaceAll('_',' ');const status=s=>'<span class="status '+s+'">'+label(s)+'</span>';
async function ensureSession(){if(sessionStorage.getItem('bravo_token'))return;const r=await fetch('/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:'dashboard@bravoorders.com',password:'dashboard-session'})});if(!r.ok)throw new Error('auth');sessionStorage.setItem('bravo_token',(await r.json()).token)}
async function api(path,options={}){const h={'Content-Type':'application/json',...(options.headers||{})},t=sessionStorage.getItem('bravo_token');if(t)h.Authorization='Bearer '+t;const r=await fetch(API_BASE+path,{...options,headers:h});if(!r.ok)throw new Error(String(r.status));return r.json()}
async function loadLive(){try{await ensureSession();const d=await Promise.all([api('/api/orders').catch(()=>[]),api('/api/inventory/products').catch(()=>[]),api('/api/shipping/shipments').catch(()=>[]),api('/api/payments/ledger').catch(()=>[])]);state={orders:d[0]||[],products:d[1]||[],shipments:d[2]||[],ledger:d[3]||[],source:'live'};renderAll()}catch(e){state.source='unavailable';renderAll();console.error('Bravo API unavailable',e)}}function renderKpis(){const active=state.orders.filter(o=>!['delivered','cancelled'].includes(o.status)).length,rev=state.orders.filter(o=>o.status!=='cancelled').reduce((a,o)=>a+Number(o.total||0),0),low=state.products.filter(p=>Number(p.available??(p.stock-p.reserved))<10).length;$('#kpiGrid').innerHTML=[['Orders',state.orders.length,'▣'],['Active fulfillment',active,'⌁'],['Net revenue',money(rev),'¤'],['Low stock',low,'◇']].map(x=>'<article class="kpi-card"><div class="kpi-top"><span class="kpi-label">'+x[0]+'</span><span class="kpi-icon">'+x[2]+'</span></div><div class="kpi-value">'+x[1]+'</div><div class="kpi-foot">'+(state.source==='live'?'Live production data':'Waiting for live API')+'</div></article>').join('')}
function orderRow(o){return '<tr data-order-id="'+o.id+'"><td class="order-id">#'+o.id+'</td><td>'+o.customer_id+'</td><td>'+status(o.status)+'</td><td>'+(o.items||[]).reduce((a,i)=>a+Number(i.quantity||0),0)+'</td><td><b>'+money(o.total,o.currency)+'</b></td><td>'+new Date(o.created_at).toLocaleString()+'</td><td>›</td></tr>'}

function openOrder(id){
  const order=state.orders.find(o=>Number(o.id)===Number(id));
  if(!order)return;

  const shipment=state.shipments.find(s=>Number(s.order_id)===Number(order.id));
  const itemCount=(order.items||[]).reduce((a,i)=>a+Number(i.quantity||0),0);
  let orderActionContent='';
  if(['pending','confirmed','processing'].includes(order.status)){
    orderActionContent='<button id="cancelOrderBtn" class="btn btn-secondary">Cancel Order</button>';
    if(order.status==='processing'&&shipment)orderActionContent+='<button id="markInTransitBtn" class="btn btn-secondary" style="margin-left: 8px">Mark In Transit</button>';
  }else if(order.status==='shipped'){
    orderActionContent='<p>This order can no longer be cancelled because it has been shipped.</p>'+
      (shipment?'<button id="markDeliveredBtn" class="btn btn-secondary" style="margin-left: 8px">Mark Delivered</button>':'');
  }else if(order.status==='delivered'){
    orderActionContent='<p>This order has been delivered and can no longer be modified.</p>';
  }else if(order.status==='cancelled'){
    orderActionContent='<p>This order has been cancelled and can no longer be modified.</p>';
  }
  const orderAction=orderActionContent;

  $('#drawerTitle').textContent='Order #'+order.id;

  $('#drawerContent').innerHTML=
    '<div class="detail-block">'+
      '<h3>ORDER INFORMATION</h3>'+
      '<div class="detail-grid">'+
        '<div class="detail-field"><label>Status</label><strong>'+label(order.status)+'</strong></div>'+
        '<div class="detail-field"><label>Customer</label><strong>'+order.customer_id+'</strong></div>'+
        '<div class="detail-field"><label>Items</label><strong>'+itemCount+'</strong></div>'+
        '<div class="detail-field"><label>Total</label><strong>'+money(order.total,order.currency)+'</strong></div>'+
        '<div class="detail-field"><label>Created</label><strong>'+new Date(order.created_at).toLocaleString()+'</strong></div>'+
        '<div class="detail-field"><label>Currency</label><strong>'+(order.currency||'GBP')+'</strong></div>'+
      '</div>'+
    '</div>'+
    '<div class="detail-block">'+
      '<h3>SHIPMENT</h3>'+
      (shipment?
        '<div class="detail-grid">'+
          '<div class="detail-field"><label>Carrier</label><strong>'+(shipment.carrier||'—')+'</strong></div>'+
          '<div class="detail-field"><label>Status</label><strong>'+label(shipment.status)+'</strong></div>'+
          '<div class="detail-field"><label>Tracking</label><strong>'+(shipment.tracking_number||'—')+'</strong></div>'+
          '<div class="detail-field"><label>Destination</label><strong>'+((shipment.city||'—')+', '+(shipment.country||''))+'</strong></div>'+
        '</div>'
        :
        '<p>No shipment has been created for this order yet.</p>'
      )+
    '</div>'+
    '<div class="detail-block">'+
      '<h3>ORDER ACTIONS</h3>'+
      orderAction+
    '</div>';

  $('#orderDrawer').classList.add('open');
  $('#drawerBackdrop').classList.add('open');
  if($('#cancelOrderBtn'))$('#cancelOrderBtn').onclick=async()=>{
    try{
      await api('/api/orders/status',{method:'PUT',body:JSON.stringify({order_id:Number(id),new_status:'cancelled'})});
      await loadLive();
      openOrder(id);
      toast('Order cancelled');
    }catch(error){
      console.error(error);
      toast('Could not cancel order');
    }
  };
  if($('#markInTransitBtn'))$('#markInTransitBtn').onclick=async()=>{
    try{
      await api('/api/shipping/webhook',{method:'POST',body:JSON.stringify({tracking_number:shipment.tracking_number,status:'in_transit',location:'London',description:'Shipment is in transit'})});
      await loadLive();
      openOrder(id);
      toast('Shipment marked in transit');
    }catch(error){
      console.error(error);
      toast('Could not update shipment');
    }
  };
  if($('#markDeliveredBtn'))$('#markDeliveredBtn').onclick=async()=>{
    try{
      await api('/api/shipping/webhook',{method:'POST',body:JSON.stringify({tracking_number:shipment.tracking_number,status:'delivered',location:'London',description:'Shipment delivered'})});
      await loadLive();
      openOrder(id);
      toast('Shipment marked delivered');
    }catch(error){
      console.error(error);
      toast('Could not update shipment');
    }
  };
}

function renderOrders(filter='all',q=''){const a=state.orders.filter(o=>(filter==='all'||o.status===filter)&&String(o.id+' '+o.customer_id).toLowerCase().includes(q.toLowerCase()));$('#ordersTable').innerHTML=a.map(orderRow).join('')||'<tr><td colspan="7">No orders found.</td></tr>';$('#recentOrders').innerHTML=state.orders.slice(0,5).map(orderRow).join('')}
function renderInventory(){const av=p=>Number(p.available??(p.stock-p.reserved)),q=($('#inventorySearch')?.value||'').toLowerCase();$('#inventoryKpis').innerHTML=[['Units on hand',state.products.reduce((a,p)=>a+Number(p.stock||0),0)],['Units reserved',state.products.reduce((a,p)=>a+Number(p.reserved||0),0)],['Low-stock products',state.products.filter(p=>av(p)<10).length]].map(x=>'<div class="mini-card"><label>'+x[0]+'</label><strong>'+x[1]+'</strong></div>').join('');$('#inventoryTable').innerHTML=state.products.filter(p=>String(p.name+' '+p.sku).toLowerCase().includes(q)).map(p=>'<tr><td><b>'+p.name+'</b></td><td>'+p.sku+'</td><td>'+p.stock+'</td><td>'+p.reserved+'</td><td><b>'+av(p)+'</b></td><td>'+status(av(p)<10?'pending':'delivered')+'</td></tr>').join('')}
function renderShipments(){$('#shippingKpis').innerHTML='<div class="mini-card"><label>Active shipments</label><strong>'+state.shipments.filter(s=>s.status!=='delivered').length+'</strong></div>';$('#shipmentsTable').innerHTML=state.shipments.map(s=>'<tr><td><b>#'+s.id+'</b></td><td>#'+s.order_id+'</td><td>'+s.carrier+'</td><td>'+s.tracking_number+'</td><td>'+status(s.status)+'</td><td>'+(s.city||'')+', '+(s.country||'')+'</td><td>'+(s.estimated_delivery||'—')+'</td></tr>').join('')}
function renderPayments(){const cr=state.ledger.reduce((a,e)=>a+Number(e.credit||0),0),dr=state.ledger.reduce((a,e)=>a+Number(e.debit||0),0);$('#paymentKpis').innerHTML=[['Gross credits',money(cr)],['Gross debits',money(dr)],['Net ledger',money(cr-dr)]].map(x=>'<div class="mini-card"><label>'+x[0]+'</label><strong>'+x[1]+'</strong></div>').join('');$('#paymentsTable').innerHTML=state.ledger.map(e=>'<tr><td><b>'+e.payment_id+'</b></td><td>'+label(e.entry_type)+'</td><td>'+e.description+'</td><td>'+(e.debit?money(e.debit,e.currency):'—')+'</td><td>'+(e.credit?money(e.credit,e.currency):'—')+'</td><td>'+new Date(e.created_at).toLocaleString()+'</td></tr>').join('')}
function renderServices(){const s=[['dashboard-api','Public UI & analytics','8086'],['api-gateway','Internal routing & auth','8080'],['order-service','Order lifecycle','8081'],['inventory-service','Stock & reservations','8082'],['payment-service','Payments & ledger','8083'],['notification-service','Email & SMS','8084'],['shipping-service','Shipping & tracking','8085'],['worker','SQS orchestration','—'],['scheduler','Scheduled jobs','—']];$('#serviceGrid').innerHTML=s.map((x,i)=>'<article class="service-card"><div class="service-top"><span class="service-icon">'+(i+1)+'</span><span class="service-state">● Configured</span></div><h3>'+x[0]+'</h3><p>'+x[1]+'</p><div class="service-meta"><div><span>Port</span><b>'+x[2]+'</b></div></div></article>').join('')}
function renderOrderCounts(){$$('#orderTabs button').forEach(b=>{const n=b.dataset.status==='all'?state.orders.length:state.orders.filter(o=>o.status===b.dataset.status).length;const counters=Array.from(b.querySelectorAll('*')).filter(e=>/^\D*\d+\D*$/.test(e.textContent.trim()));if(counters.length)counters[counters.length-1].textContent=counters[counters.length-1].textContent.replace(/\d+/,n);else{const text=Array.from(b.childNodes).find(node=>node.nodeType===Node.TEXT_NODE&&/\d+/.test(node.textContent));if(text)text.textContent=text.textContent.replace(/\d+/,n)}})}
function renderAll(){renderOrderCounts();renderKpis();renderOrders();renderInventory();renderShipments();renderPayments();renderServices();$('#orderChart').innerHTML=[31,44,38,52,41,57,48].map((v,i)=>'<div class="bar-col"><div class="bar" style="height:'+(v/60*88)+'%"></div><span>'+['Fri','Sat','Sun','Mon','Tue','Wed','Today'][i]+'</span></div>').join('');$('#flowList').innerHTML=['pending','confirmed','processing','shipped','delivered'].map(s=>'<div class="flow-row"><label>'+s+'</label><div class="progress"><span style="width:'+Math.max(8,state.orders.filter(o=>o.status===s).length*20)+'%"></span></div><b>'+state.orders.filter(o=>o.status===s).length+'</b></div>').join('');$('#alertsList').innerHTML=state.products.filter(p=>Number(p.available??(p.stock-p.reserved))<10).map(p=>'<div class="alert-item"><span class="alert-icon">!</span><div class="alert-copy"><strong>'+p.name+'</strong><p>Low stock needs attention</p></div></div>').join('')||'<div class="alert-item"><div class="alert-copy"><strong>No low-stock alerts</strong></div></div>';$('#productSelect').innerHTML=state.products.map(p=>'<option value="'+p.id+'">'+p.name+' — '+money(p.price)+'</option>').join('')}

function switchView(n, updateHash = true) {
  const view = $('#' + n + '-view');

  if (!view) {
    n = 'overview';
  }

  $$('.view').forEach(v=>v.classList.remove('active'));
  $('#' + n + '-view').classList.add('active');
  $$('.nav-item').forEach(x=>x.classList.toggle('active',x.dataset.view===n));
  $('#crumb').textContent=n[0].toUpperCase()+n.slice(1);
  $('#sidebar').classList.remove('mobile-open');

  if (updateHash && window.location.hash !== '#' + n) {
    window.location.hash = n;
  }
}

function loadViewFromHash() {
  const view = window.location.hash.replace('#', '') || 'overview';
  switchView(view, false);
}

window.addEventListener('hashchange', loadViewFromHash);

function toast(m){const t=$('#toast');t.textContent=m;t.classList.add('show');setTimeout(()=>t.classList.remove('show'),2200)}
$$('.nav-item').forEach(n=>n.onclick=()=>switchView(n.dataset.view));$$('[data-jump]').forEach(b=>b.onclick=()=>switchView(b.dataset.jump));$('#collapseBtn').onclick=()=>$('#sidebar').classList.toggle('collapsed');$('#mobileMenu').onclick=()=>$('#sidebar').classList.toggle('mobile-open');$('#refreshBtn').onclick=async()=>{await loadLive();toast(state.source==='live'?'Live data refreshed':'API unavailable')};$('#drawerBackdrop').onclick=$('#closeDrawer').onclick=()=>{$('#orderDrawer').classList.remove('open');$('#drawerBackdrop').classList.remove('open')};const openModal=()=>$('#modalBackdrop').classList.add('open'),closeModal=()=>$('#modalBackdrop').classList.remove('open');$('#newOrderBtn').onclick=openModal;$('#newOrderBtn2').onclick=openModal;$('#closeModal').onclick=closeModal;$('#cancelModal').onclick=closeModal;$('#modalBackdrop').onclick=e=>{if(e.target.id==='modalBackdrop')closeModal()};$('#orderForm').onsubmit=async e=>{e.preventDefault();const p=state.products.find(x=>x.id===$('#productSelect').value);if(!p)return;try{await api('/api/orders',{method:'POST',body:JSON.stringify({items:[{product_id:p.id,quantity:Number($('#quantityInput').value),price:p.price}],currency:$('#currencyInput').value,notes:$('#notesInput').value,recipient_name:$('#recipientNameInput').value,address_line1:$('#addressInput').value,address_line2:"",city:$('#cityInput').value,postcode:$('#postcodeInput').value,country:$('#countryInput').value,weight_kg:1.0})});closeModal();await loadLive();toast('Order created')}catch{toast('Could not create order')}};$('#orderSearch').oninput=()=>renderOrders($('#orderTabs .active').dataset.status,$('#orderSearch').value);$$('#orderTabs button').forEach(b=>b.onclick=()=>{$$('#orderTabs button').forEach(x=>x.classList.remove('active'));b.classList.add('active');renderOrders(b.dataset.status,$('#orderSearch').value)});$('#inventorySearch').oninput=renderInventory;$('#addProductBtn').onclick=()=>toast('Inventory creation remains API-managed');$('#globalSearch').onkeydown=e=>{if(e.key==='Enter'){switchView('orders');$('#orderSearch').value=e.target.value;renderOrders('all',e.target.value)}};

$('#ordersTable').onclick=e=>{
  const row=e.target.closest('tr[data-order-id]');
  if(!row)return;
  openOrder(row.dataset.orderId);
};

$('#recentOrders').onclick=e=>{
  const row=e.target.closest('tr[data-order-id]');
  if(!row)return;
  openOrder(row.dataset.orderId);
};

loadViewFromHash();
renderAll();
loadLive();