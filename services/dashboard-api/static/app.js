const API_BASE='';
const ADMIN_UI_SESSION='bravo_admin_session';
const CUSTOMER_UI_SESSION='bravo_customer_session';
const CUSTOMER_BASKET='bravo_customer_basket';
const $=s=>document.querySelector(s),$$=s=>Array.from(document.querySelectorAll(s));let state={orders:[],products:[],shipments:[],ledger:[],source:'loading'},dashboardStarted=false;
let customerProducts=[],customerProductsLoaded=false,customerProductsLoading=false,customerProductsError='',customerBasket=readCustomerBasket(),checkoutSubmitting=false,customerConfirmation=readCustomerConfirmation(),customerOrderSummaries=readCustomerOrderSummaries(),customerOrders=[],customerOrdersLoading=false,customerOrdersError='',selectedCustomerOrderId=null,customerPayments=[],customerPaymentsUnavailable=false,customerRefundRequests=[],customerRefundRequestsUnavailable=false,customerRefundFormOrderId=null,customerRefundSubmitting=false,adminRefundRequests=[],adminRefundRequestsLoading=false,adminRefundRequestsError=false,adminRefundActions=new Set();
const money=(n,c='GBP')=>new Intl.NumberFormat('en-GB',{style:'currency',currency:c}).format(Number(n||0));const label=s=>(s||'').replaceAll('_',' ');const status=s=>'<span class="status '+s+'">'+label(s)+'</span>';
function readCustomerBasket(){try{const basket=JSON.parse(sessionStorage.getItem(CUSTOMER_BASKET)||'[]');return Array.isArray(basket)?basket.filter(i=>i&&i.id!==undefined&&Number(i.quantity)>0).map(i=>({...i,quantity:Math.max(1,Math.floor(Number(i.quantity)||1))})):[]}catch(error){console.error('Could not read customer basket',error);return []}}
function readCustomerConfirmation(){try{return JSON.parse(sessionStorage.getItem('bravo_customer_confirmation')||'null')}catch(error){console.error('Could not read order confirmation',error);return null}}
function readCustomerOrderSummaries(){try{const summaries=JSON.parse(sessionStorage.getItem('bravo_customer_order_summaries')||'{}');return summaries&&typeof summaries==='object'&&!Array.isArray(summaries)?summaries:{}}catch(error){console.error('Could not read customer order summaries',error);return {}}}
function saveCustomerBasket(){sessionStorage.setItem(CUSTOMER_BASKET,JSON.stringify(customerBasket));renderCustomerBasketCount()}
function escapeHtml(value){return String(value??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
function availableStock(product){const stock=Number(product.available??(Number(product.stock||0)-Number(product.reserved||0)));return Number.isFinite(stock)?Math.max(0,Math.floor(stock)):0}
async function api(path,options={}){const h={'Content-Type':'application/json',...(options.headers||{})},t=sessionStorage.getItem('bravo_token');if(t)h.Authorization='Bearer '+t;const r=await fetch(API_BASE+path,{...options,headers:h});if(!r.ok)throw new Error(String(r.status));return r.json()}
async function loadLive(){try{const d=await Promise.all([api('/api/orders?all=true').catch(()=>[]),api('/api/inventory/products').catch(()=>[]),api('/api/shipping/shipments').catch(()=>[]),api('/api/payments/ledger').catch(()=>[])]);state={orders:d[0]||[],products:d[1]||[],shipments:d[2]||[],ledger:d[3]||[],source:'live'};renderAll()}catch(e){state.source='unavailable';renderAll();console.error('Bravo API unavailable',e)}}function renderKpis(){const active=state.orders.filter(o=>!['delivered','cancelled'].includes(o.status)).length,rev=state.orders.filter(o=>o.status!=='cancelled').reduce((a,o)=>a+Number(o.total||0),0),low=state.products.filter(p=>Number(p.available??(p.stock-p.reserved))<10).length;$('#kpiGrid').innerHTML=[['Orders',state.orders.length,'▣'],['Active fulfillment',active,'⌁'],['Net revenue',money(rev),'¤'],['Low stock',low,'◇']].map(x=>'<article class="kpi-card"><div class="kpi-top"><span class="kpi-label">'+x[0]+'</span><span class="kpi-icon">'+x[2]+'</span></div><div class="kpi-value">'+x[1]+'</div><div class="kpi-foot">'+(state.source==='live'?'Live production data':'Waiting for live API')+'</div></article>').join('')}
function renderCustomerBasketCount(){$('#basketCount').textContent=customerBasket.reduce((total,item)=>total+Number(item.quantity||0),0)}
function renderCustomerProducts(){
  const grid=$('#customerProductGrid'),message=$('#customerProductMessage');
  if(customerProductsError){
    message.textContent=customerProductsError;
    message.hidden=false;
    grid.innerHTML='<div class="customer-empty"><h2>We couldn’t load the shop.</h2><p>Please check your connection and try again.</p><button class="secondary-btn" id="retryProductsBtn" type="button">Try again</button></div>';
    return;
  }
  message.hidden=true;
  if(!customerProducts.length){
    grid.innerHTML='<div class="customer-empty"><span aria-hidden="true">◇</span><h2>No products available just yet.</h2><p>Please check back soon.</p></div>';
    return;
  }
  grid.innerHTML=customerProducts.map(product=>{
    const stock=availableStock(product),id=escapeHtml(product.id),name=escapeHtml(product.name||'Product'),sku=product.sku?'<span class="product-sku">SKU '+escapeHtml(product.sku)+'</span>':'',image=productImages[product.sku] || '';

return '<article class="customer-product-card"><div class="product-art">'+(image?'<img src="'+image+'" alt="'+name+'" class="product-image">':'<span>B</span><i>BRAVO<br>ORDERS</i>')+'</div><div class="product-card-copy">'+sku+'<h2>'+name+'</h2><div class="product-price">'+money(product.price,product.currency||'GBP')+'</div><p class="product-stock">'+(stock>0?stock+' available':'Currently out of stock')+'</p><div class="product-card-actions"><label>Qty<input type="number" min="1" max="'+stock+'" value="1" data-product-quantity="'+id+'" '+(stock===0?'disabled':'')+' /></label><button class="primary-btn add-basket-btn" type="button" data-add-product="'+id+'" '+(stock===0?'disabled':'')+'>Add to basket</button></div></div></article>';
  }).join('');
}
async function loadCustomerProducts(force=false){
  if(customerProductsLoaded&&!force)return;
  if(customerProductsLoading)return;
  customerProductsLoading=true;
  customerProductsError='';
  $('#customerProductMessage').hidden=true;
  $('#customerProductGrid').innerHTML='<div class="customer-empty"><span class="customer-loading"></span><p>Loading products…</p></div>';
  try{
    const products=await api('/api/inventory/products');
    if(!Array.isArray(products))throw new Error('Invalid product catalogue response');
    customerProducts=products;
    let basketAdjusted=false;
    customerBasket.forEach(item=>{
      const product=customerProducts.find(entry=>String(entry.id)===String(item.id));
      if(!product)return;
      const available=availableStock(product);
      if(available>0&&item.quantity>available){item.quantity=available;basketAdjusted=true}
      Object.assign(item,{name:product.name,sku:product.sku,price:product.price,currency:product.currency,available});
    });
    if(basketAdjusted)saveCustomerBasket();
    customerProductsLoaded=true;
    
    const productImages = {
  'BRAVO-001': '/images/products/bravo-test-product.webp',
  'BRAVO-002': '/images/products/wireless-keyboard.webp',
  'BRAVO-003': '/images/products/usb-c-hub.webp',
  'BRAVO-004': '/images/products/laptop-stand.webp',
  'BRAVO-005': '/images/products/wireless-mouse.webp',
  'BRAVO-006': '/images/products/desk-lamp.webp'

};
    renderCustomerProducts();
    renderCustomerBasket();
    if(window.location.hash==='#customer-checkout')renderCheckout();
    if(basketAdjusted){
      $('#basketUpdateMessage').textContent='Your basket was updated to reflect current product availability.';
      $('#basketUpdateMessage').hidden=false;
    }
  }catch(error){
    customerProductsError='The product catalogue is temporarily unavailable.';
    customerProductsLoaded=false;
    console.error('Could not load customer products',error);
    renderCustomerProducts();
  }finally{
    customerProductsLoading=false;
  }
}
function renderCustomerBasket(){
  renderCustomerBasketCount();
  const container=$('#customerBasketItems');
  if(!customerBasket.length){
    container.innerHTML='<div class="customer-empty"><span aria-hidden="true">▢</span><h2>Your basket is waiting.</h2><p>Browse the shop and add products to get started.</p><button class="primary-btn" type="button" data-customer-view="shop">Browse the shop</button></div>';
    $('#basketSubtotal').textContent=money(0);
    $('#checkoutBtn').disabled=true;
    return;
  }
  let subtotal=0;
  container.innerHTML=customerBasket.map(item=>{
    const product=customerProducts.find(p=>String(p.id)===String(item.id))||item;
    const available=availableStock(product),quantity=Number(item.quantity)||1,price=Number(product.price??item.price)||0,lineTotal=price*quantity;
    subtotal+=lineTotal;
    const outOfStock=available===0;
    const name=escapeHtml(product.name||'Product');
    return '<article class="basket-item"><div class="basket-item-art" aria-hidden="true">B</div><div class="basket-item-info"><span class="product-sku">'+(product.sku?'SKU '+escapeHtml(product.sku):'Bravo Orders product')+'</span><h2>'+name+'</h2><p>'+money(price,product.currency||item.currency||'GBP')+' each'+(outOfStock?' · Currently unavailable':'')+'</p><div class="basket-quantity"><button type="button" data-basket-action="decrease" data-product-id="'+escapeHtml(item.id)+'" aria-label="Decrease quantity" '+(quantity<=1?'disabled':'')+'>−</button><span>'+quantity+'</span><button type="button" data-basket-action="increase" data-product-id="'+escapeHtml(item.id)+'" aria-label="Increase quantity" '+(outOfStock||quantity>=available?'disabled':'')+'>+</button></div></div><strong class="basket-line-total">'+money(lineTotal,product.currency||item.currency||'GBP')+'</strong><button class="remove-basket-item" type="button" data-basket-action="remove" data-product-id="'+escapeHtml(item.id)+'">Remove</button></article>';
  }).join('');
  $('#basketSubtotal').textContent=money(subtotal);
  $('#checkoutBtn').disabled=false;
}
function addCustomerProduct(id){
  const product=customerProducts.find(p=>String(p.id)===String(id));
  if(!product)return;
  const available=availableStock(product),input=Array.from($$('#customerProductGrid [data-product-quantity]')).find(element=>element.dataset.productQuantity===String(id)),requested=Math.max(1,Math.floor(Number(input?.value)||1));
  if(available<1){$('#customerProductMessage').textContent='This product is currently out of stock.';$('#customerProductMessage').hidden=false;return}
  const current=customerBasket.find(item=>String(item.id)===String(id)),quantity=Math.min(available,requested+(current?.quantity||0));
  if(current)current.quantity=quantity;
  else customerBasket.push({id:product.id,name:product.name,sku:product.sku,price:product.price,currency:product.currency,available,quantity});
  saveCustomerBasket();
  renderCustomerBasket();
  $('#customerProductMessage').textContent='Added to your basket.';
  $('#customerProductMessage').hidden=false;
}
function changeCustomerBasket(id,action){
  const item=customerBasket.find(entry=>String(entry.id)===String(id));
  if(!item)return;
  if(action==='remove')customerBasket=customerBasket.filter(entry=>String(entry.id)!==String(id));
  if(action==='increase'){
    const product=customerProducts.find(entry=>String(entry.id)===String(id))||item;
    if(item.quantity<availableStock(product))item.quantity++;
  }
  if(action==='decrease'&&item.quantity>1)item.quantity--;
  saveCustomerBasket();
  renderCustomerBasket();
}
function renderCheckout(){
  let subtotal=0;
  $('#checkoutSummaryItems').innerHTML=customerBasket.map(item=>{
    const product=customerProducts.find(entry=>String(entry.id)===String(item.id))||item,quantity=Number(item.quantity)||1,price=Number(product.price??item.price)||0,total=price*quantity;
    subtotal+=total;
    return '<div class="checkout-summary-item"><div><strong>'+escapeHtml(product.name||'Product')+'</strong><span>'+quantity+' × '+money(price,product.currency||item.currency||'GBP')+'</span></div><b>'+money(total,product.currency||item.currency||'GBP')+'</b></div>';
  }).join('');
  $('#checkoutSubtotal').textContent=money(subtotal);
  $('#checkoutTotal').textContent=money(subtotal);
  $('#checkoutError').hidden=true;
}
function renderOrderConfirmation(){
  if(!customerConfirmation)return;
  const order=customerConfirmation.order||{},summary=customerConfirmation.summary||{},identifier=order.id??order.order_id??order.order_number??customerConfirmation.id??customerConfirmation.order_id??customerConfirmation.order_number,statusValue=order.status??customerConfirmation.status,totalInfo=customerOrderTotal(order,summary),currency=totalInfo?.currency||order.currency||summary.currency||'GBP',items=enrichOrderItems(order.items||[],summary.items||[]);
  $('#confirmationDetails').innerHTML=(identifier!==undefined&&identifier!==null?'<div><span>Order number</span><strong>#'+escapeHtml(identifier)+'</strong></div>':'')+(statusValue?'<div><span>Status</span><strong>'+escapeHtml(customerStatusLabel(statusValue))+'</strong></div>':'')+(items.length?'<div class="confirmation-items"><span>Items</span><strong>'+items.map(item=>{const quantity=Number(item.quantity||0),price=Number(item.unit_price??item.price),line=Number.isFinite(price)?' — '+money(Number(item.line_total??price*quantity),item.currency||currency):'';return escapeHtml(item.product_name||item.name||item.product_id||'Item')+' × '+quantity+line}).join('<br>')+'</strong></div>':'')+(totalInfo!==null?'<div><span>Order total</span><strong>'+money(totalInfo.amount,totalInfo.currency)+'</strong></div>':'<div><span>Order total</span><strong>Total unavailable</strong></div>');
}
function orderIdentifier(order){return order.id??order.order_id??order.order_number}
function orderSummaryFor(order){const id=orderIdentifier(order);return id===undefined||id===null?null:customerOrderSummaries[String(id)]||null}
function customerOrderTotal(order,providedSummary=null){
  const summary=providedSummary||orderSummaryFor(order);
  const backendTotal=order.total;
  if(backendTotal!==undefined&&backendTotal!==null&&backendTotal!==''&&Number.isFinite(Number(backendTotal))&&!(Number(backendTotal)===0&&summary&&Number(summary.total)>0))return {amount:Number(backendTotal),currency:order.currency||'GBP'};
  if(summary&&summary.total!==undefined&&summary.total!==null&&summary.total!==''&&Number.isFinite(Number(summary.total)))return {amount:Number(summary.total),currency:order.currency||summary.currency||'GBP'};
  return null;
}
function customerStatusLabel(value){
  return ({pending:'Pending',confirmed:'Confirmed',processing:'Processing',shipped:'Shipped',delivered:'Delivered',cancelled:'Cancelled'})[String(value||'').toLowerCase()]||label(value||'Status unavailable');
}
function customerStatusClass(value){
  const statusValue=String(value||'').toLowerCase();
  return ['pending','confirmed','processing','shipped','delivered','cancelled'].includes(statusValue)?statusValue:'pending';
}
function customerOrderItems(order){
  const summary=orderSummaryFor(order);
  return enrichOrderItems(order.items||[],summary?.items||[]);
}
function enrichOrderItems(items,summaryItems){
  if(!items.length)return summaryItems;
  return items.map(item=>{
    const id=item.product_id??item.id,summary=summaryItems.find(entry=>String(entry.product_id??entry.id)===String(id))||{};
    const backendPrice=item.unit_price??item.price,savedPrice=summary.unit_price??summary.price;
    const price=backendPrice!==undefined&&backendPrice!==null&&Number.isFinite(Number(backendPrice))&&!(Number(backendPrice)===0&&Number(savedPrice)>0)?Number(backendPrice):savedPrice;
    const backendLineTotal=item.line_total,summaryLineTotal=summary.line_total;
    const lineTotal=backendLineTotal!==undefined&&backendLineTotal!==null&&Number.isFinite(Number(backendLineTotal))&&!(Number(backendLineTotal)===0&&Number(summaryLineTotal)>0)?Number(backendLineTotal):summaryLineTotal;
    return {...summary,...item,name:item.product_name||item.name||summary.name,price,unit_price:price,line_total:lineTotal,currency:item.currency||summary.currency};
  });
}
function orderItemCount(items){return items.reduce((count,item)=>count+Number(item.quantity||0),0)}
function formatCustomerOrderDate(value){
  if(!value)return 'Date unavailable';
  const date=new Date(value);
  return Number.isNaN(date.getTime())?'Date unavailable':date.toLocaleString();
}
function customerPaymentForOrder(order){
  const id=orderIdentifier(order);
  if(id===undefined||id===null||customerPaymentsUnavailable)return null;
  return customerPayments.find(payment=>payment&&String(payment.order_id)===String(id)&&payment.customer_id==='customer@bravoorders.com'&&payment.id&&String(payment.method||'').toLowerCase()!=='refund'&&Number.isFinite(Number(payment.amount))&&Number(payment.amount)>0&&['completed','partially_refunded'].includes(String(payment.status||'').toLowerCase()))||null;
}
function customerRefundForOrder(order){
  const id=orderIdentifier(order);
  if(id===undefined||id===null||customerRefundRequestsUnavailable)return null;
  const matches=customerRefundRequests.filter(request=>request&&request.customer_id==='customer@bravoorders.com'&&String(request.order_id)===String(id));
  return matches.find(request=>String(request.status||'').toLowerCase()==='refunded')||matches.find(request=>String(request.status||'').toLowerCase()==='requested')||matches.find(request=>String(request.status||'').toLowerCase()==='approved')||matches.sort((a,b)=>new Date(b.created_at||0)-new Date(a.created_at||0))[0]||null;
}
function customerRefundStatusLabel(value){
  return ({requested:'Refund requested — awaiting review',approved:'Refund approved — processing',rejected:'Refund request rejected',refunded:'Refund completed'})[String(value||'').toLowerCase()]||'Refund status unavailable';
}
function renderCustomerOrderDetail(order){
  if(!order){$('#customerOrderDetail').hidden=true;return}
  const id=orderIdentifier(order),items=customerOrderItems(order),total=customerOrderTotal(order),statusValue=order.status;
  const payment=customerPaymentForOrder(order),refund=customerRefundForOrder(order),refundStatus=refund?String(refund.status||'').toLowerCase():'',hasActiveRefund=['requested','approved'].includes(refundStatus),hasCompletedRefund=refundStatus==='refunded';
  const tracker=['pending','confirmed','processing','shipped','delivered'],steps=['Order placed','Confirmed','Processing','Shipped','Delivered'],current=tracker.indexOf(String(statusValue||'').toLowerCase());
  const lifecycle=String(statusValue||'').toLowerCase()==='cancelled'
    ?'<div class="order-cancelled-state"><span>×</span><strong>Cancelled</strong></div>'
    :'<ol class="order-lifecycle">'+tracker.map((step,index)=>'<li class="'+(current>=index?'complete':'')+(current===index?' current':'')+'"><span class="lifecycle-mark">'+(current>index?'✓':index+1)+'</span><div><strong>'+steps[index]+'</strong></div></li>').join('')+'</ol>';
  let refundContent='';
  if(customerRefundRequestsUnavailable)refundContent='<p class="refund-info-message">Refund status could not be checked, so a new request is unavailable. Refresh order status to try again.</p>';
  else if(refundStatus)refundContent='<p class="refund-state-message '+escapeHtml(refundStatus)+'">'+escapeHtml(customerRefundStatusLabel(refundStatus))+'</p>';
  if(!customerPaymentsUnavailable&&!customerRefundRequestsUnavailable&&payment&&String(payment.status||'').toLowerCase()==='completed'&&!hasActiveRefund&&!hasCompletedRefund){
    refundContent+=(refundStatus==='rejected'?'<p class="refund-info-message">'+escapeHtml(customerRefundStatusLabel(refundStatus))+'</p>':'')+
      (customerRefundFormOrderId===String(id)
        ?'<form class="customer-refund-form" id="customerRefundForm"><label>Order number<input value="#'+escapeHtml(id)+'" readonly /></label><label>Refund amount<input id="customerRefundAmount" type="number" min="0.01" max="'+Number(payment.amount)+'" step="0.01" value="'+Number(payment.amount)+'" readonly /></label><label>Reason<textarea id="customerRefundReason" rows="3" maxlength="500" required placeholder="Tell us why you are requesting a refund"></textarea></label><p class="refund-form-error" id="customerRefundError" hidden></p><div class="refund-form-actions"><button class="secondary-btn" id="cancelCustomerRefundBtn" type="button" '+(customerRefundSubmitting?'disabled':'')+'>Cancel</button><button class="primary-btn" id="submitCustomerRefundBtn" type="submit" '+(customerRefundSubmitting?'disabled':'')+'>'+(customerRefundSubmitting?'Submitting request...':'Submit refund request')+'</button></div></form>'
        :'<button class="secondary-btn customer-request-refund" id="requestCustomerRefundBtn" type="button">Request refund</button>');
  }else if(customerPaymentsUnavailable)refundContent+='<p class="refund-info-message">Payment information is temporarily unavailable. Refresh order status to try again.</p>';
  else if(payment&&String(payment.status||'').toLowerCase()==='partially_refunded'&&!refundStatus)refundContent+='<p class="refund-info-message">A previous refund has been processed. The remaining refundable amount is unavailable.</p>';
  else if(!payment&&!refundStatus)refundContent+='<p class="refund-info-message">A refund cannot be requested because eligible payment information is unavailable.</p>';
  $('#customerOrderDetail').hidden=false;
  $('#customerOrderDetail').innerHTML='<div class="customer-order-detail-head"><div><p class="eyebrow">ORDER DETAILS</p><h2>'+(id!==undefined&&id!==null?'Order #'+escapeHtml(id):'Order')+'</h2></div><button class="secondary-btn" type="button" id="refreshSelectedOrderBtn">↻ Refresh status</button></div><div class="customer-order-detail-meta"><span class="customer-status '+customerStatusClass(statusValue)+'">'+escapeHtml(customerStatusLabel(statusValue))+'</span><span>'+escapeHtml(formatCustomerOrderDate(order.created_at||order.updated_at))+'</span></div><h3>Delivery progress</h3>'+lifecycle+'<h3>Items</h3><div class="customer-order-lines">'+(items.length?items.map(item=>{const quantity=Number(item.quantity||0),unitPrice=item.unit_price??item.price,lineTotal=item.line_total??(unitPrice!==undefined&&unitPrice!==null?Number(unitPrice)*quantity:null),priceDisplay=lineTotal!==undefined&&lineTotal!==null&&Number.isFinite(Number(lineTotal))?money(Number(lineTotal),item.currency||order.currency||'GBP'):'Price unavailable';return '<div><span>'+escapeHtml(item.product_name||item.name||item.product_id||'Item')+' × '+quantity+'</span><strong>'+priceDisplay+'</strong></div>'}).join(''):'<p>Item details unavailable.</p>')+'</div><div class="customer-order-total"><span>Order total</span><strong>'+(total!==null?money(total.amount,total.currency):'Total unavailable')+'</strong></div><section class="customer-refund-section"><h3>Refund status</h3>'+refundContent+'</section>';
  $('#customerOrderDetail').onclick=event=>{
    if(event.target.closest('#refreshSelectedOrderBtn')){loadCustomerOrders(id);return}
    if(event.target.closest('#requestCustomerRefundBtn')){customerRefundFormOrderId=String(id);renderCustomerOrders();return}
    if(event.target.closest('#cancelCustomerRefundBtn')&&!customerRefundSubmitting){customerRefundFormOrderId=null;renderCustomerOrders()}
  };
  const form=$('#customerRefundForm');
  if(form)form.onsubmit=event=>submitCustomerRefund(event,order,payment);
}
function renderCustomerOrders(){
  const list=$('#customerOrderList'),message=$('#customerOrdersMessage');
  if(customerOrdersLoading){
    message.hidden=true;
    list.innerHTML='<div class="customer-empty"><span class="customer-loading"></span><p>Loading your orders…</p></div>';
    $('#customerOrderDetail').hidden=true;
    return;
  }
  if(customerOrdersError){
    message.textContent='We couldn’t load your orders. Please try again.';
    message.hidden=false;
    list.innerHTML='<div class="customer-empty"><h2>Your orders are temporarily unavailable.</h2><p>Your order history could not be refreshed.</p><button class="secondary-btn" id="retryCustomerOrdersBtn" type="button">Try again</button></div>';
    $('#customerOrderDetail').hidden=true;
    return;
  }
  message.hidden=true;
  if(!customerOrders.length){
    list.innerHTML='<div class="customer-empty"><span aria-hidden="true">▣</span><h2>No orders yet</h2><p>Your placed orders will appear here.</p><button class="primary-btn" type="button" data-customer-view="shop">Start shopping</button></div>';
    $('#customerOrderDetail').hidden=true;
    return;
  }
  list.innerHTML=customerOrders.map(order=>{
    const id=orderIdentifier(order),items=customerOrderItems(order),total=customerOrderTotal(order),statusValue=order.status;
    return '<button class="customer-order-card '+(String(selectedCustomerOrderId)===String(id)?'selected':'')+'" type="button" data-customer-order-id="'+escapeHtml(id)+'"><span class="customer-order-card-top"><strong>'+(id!==undefined&&id!==null?'Order #'+escapeHtml(id):'Order')+'</strong><span class="customer-status '+customerStatusClass(statusValue)+'">'+escapeHtml(customerStatusLabel(statusValue))+'</span></span><span class="customer-order-card-date">'+escapeHtml(formatCustomerOrderDate(order.created_at||order.updated_at))+'</span><span class="customer-order-card-bottom"><span>'+orderItemCount(items)+' '+(orderItemCount(items)===1?'item':'items')+'</span><strong>'+(total!==null?money(total.amount,total.currency):'Total unavailable')+'</strong></span></button>';
  }).join('');
  const selected=customerOrders.find(order=>String(orderIdentifier(order))===String(selectedCustomerOrderId))||customerOrders[0];
  selectedCustomerOrderId=orderIdentifier(selected);
  $$('.customer-order-card').forEach(button=>button.classList.toggle('selected',String(button.dataset.customerOrderId)===String(selectedCustomerOrderId)));
  renderCustomerOrderDetail(selected);
}
async function submitCustomerRefund(event,order,payment){
  event.preventDefault();
  const existingRequest=customerRefundForOrder(order,payment),existingStatus=String(existingRequest?.status||'').toLowerCase();
  if(customerRefundSubmitting||!payment||String(payment.status||'').toLowerCase()!=='completed'||['requested','approved','refunded'].includes(existingStatus))return;
  const amount=Number($('#customerRefundAmount').value),reason=$('#customerRefundReason').value.trim(),button=$('#submitCustomerRefundBtn'),error=$('#customerRefundError');
  error.hidden=true;
  if(!Number.isFinite(amount)||amount<=0||amount>Number(payment.amount)||Math.round(amount*100)!==Math.round(Number(payment.amount)*100)){
    error.textContent='The refund amount must match the full payment amount.';
    error.hidden=false;
    return;
  }
  if(!reason){
    error.textContent='Please enter a reason for your refund request.';
    error.hidden=false;
    return;
  }
  customerRefundSubmitting=true;
  button.disabled=true;
  button.textContent='Submitting request...';
  try{
    await api('/api/payments/refund-requests',{method:'POST',body:JSON.stringify({payment_id:payment.id,amount,reason})});
    customerRefundFormOrderId=null;
    await loadCustomerOrders(orderIdentifier(order));
    toast('Refund requested — awaiting review');
  }catch(requestError){
    console.error('Could not submit refund request',requestError);
    await loadCustomerOrders(orderIdentifier(order));
    const refreshedRequest=customerRefundForOrder(order,payment);
    if(refreshedRequest){
      customerRefundFormOrderId=null;
      toast(customerRefundStatusLabel(refreshedRequest.status));
    }else{
      const refreshedError=$('#customerRefundError');
      if(refreshedError){refreshedError.textContent='We couldn’t submit your refund request. Please try again.';refreshedError.hidden=false}
    }
  }finally{
    customerRefundSubmitting=false;
    const submitButton=$('#submitCustomerRefundBtn');
    if(submitButton){submitButton.disabled=false;submitButton.textContent='Submit refund request'}
    const cancelButton=$('#cancelCustomerRefundBtn');
    if(cancelButton)cancelButton.disabled=false;
  }
}
async function loadCustomerOrders(selectedId=selectedCustomerOrderId){
  if(customerOrdersLoading)return;
  customerOrdersLoading=true;
  customerOrdersError='';
  renderCustomerOrders();
  try{
    const [response,paymentsResult,refundsResult]=await Promise.all([
      api('/api/orders'),
      api('/api/payments/payments').then(data=>({data})).catch(error=>({error})),
      api('/api/payments/refund-requests').then(data=>({data})).catch(error=>({error}))
    ]);
    if(!Array.isArray(response))throw new Error('Invalid orders response');
    customerOrders=response.filter(order=>order&&order.customer_id==='customer@bravoorders.com');
    customerPaymentsUnavailable=!!paymentsResult.error||!Array.isArray(paymentsResult.data);
    customerPayments=customerPaymentsUnavailable?[]:paymentsResult.data;
    customerRefundRequestsUnavailable=!!refundsResult.error||!Array.isArray(refundsResult.data);
    customerRefundRequests=customerRefundRequestsUnavailable?[]:refundsResult.data.filter(request=>request&&request.customer_id==='customer@bravoorders.com'&&customerOrders.some(order=>String(orderIdentifier(order))===String(request.order_id)));
    selectedCustomerOrderId=selectedId;
  }catch(error){
    customerOrdersError='Could not load customer orders';
    console.error('Could not load customer orders',error);
  }finally{
    customerOrdersLoading=false;
    renderCustomerOrders();
  }
}
function saveCheckoutDetails(){
  const details={name:$('#checkoutName').value,email:$('#checkoutEmail').value,address:$('#checkoutAddress').value,city:$('#checkoutCity').value,postcode:$('#checkoutPostcode').value,country:$('#checkoutCountry').value};
  sessionStorage.setItem('bravo_checkout_details',JSON.stringify(details));
}
function restoreCheckoutDetails(){
  try{
    const details=JSON.parse(sessionStorage.getItem('bravo_checkout_details')||'{}');
    $('#checkoutName').value=details.name||'';
    $('#checkoutEmail').value=details.email||'customer@bravoorders.com';
    $('#checkoutAddress').value=details.address||'';
    $('#checkoutCity').value=details.city||'';
    $('#checkoutPostcode').value=details.postcode||'';
    $('#checkoutCountry').value=details.country||'';
  }catch(error){
    console.error('Could not restore checkout details',error);
  }
}
async function placeCustomerOrder(event){
  event.preventDefault();
  if(checkoutSubmitting||!customerBasket.length)return;
  const error=$('#checkoutError'),button=$('#placeOrderBtn');
  error.hidden=true;
  checkoutSubmitting=true;
  button.disabled=true;
  button.textContent='Placing order...';
  saveCheckoutDetails();
  try{
    const summary={total:0,currency:'GBP',items:[]};
    const items=customerBasket.map(item=>{
      const product=customerProducts.find(entry=>String(entry.id)===String(item.id));
      if(!product)throw new Error('A product in your basket is no longer available. Return to your basket and refresh the shop.');
      const quantity=Number(item.quantity),available=availableStock(product);
      if(!Number.isInteger(quantity)||quantity<1||quantity>available)throw new Error('A basket quantity exceeds current availability. Return to your basket and review quantities.');
      const price=Number(product.price),currency=product.currency||'GBP';
      if(!Number.isFinite(price))throw new Error('A product price is unavailable. Return to your basket and refresh the shop.');
      const lineTotal=Math.round((price*quantity+Number.EPSILON)*100)/100;
      summary.total=Math.round((summary.total+lineTotal+Number.EPSILON)*100)/100;
      if(summary.items.length===0)summary.currency=currency;
      summary.items.push({product_id:product.id,name:product.name,sku:product.sku,unit_price:price,price,quantity,line_total:lineTotal,currency});
      return {product_id:product.id,quantity:Number(quantity),price};
    });
    const result=await api('/api/orders',{method:'POST',body:JSON.stringify({customer_id:'customer@bravoorders.com',recipient_name:$('#checkoutName').value,address_line1:$('#checkoutAddress').value,city:$('#checkoutCity').value,postcode:$('#checkoutPostcode').value,country:$('#checkoutCountry').value,items})});
    const order=result?.order||result,identifier=orderIdentifier(order);
    customerConfirmation={order,summary};
    if(identifier!==undefined&&identifier!==null){
      customerOrderSummaries[String(identifier)]={order_id:identifier,total:summary.total,currency:summary.currency,items:summary.items};
      sessionStorage.setItem('bravo_customer_order_summaries',JSON.stringify(customerOrderSummaries));
    }
    sessionStorage.setItem('bravo_customer_confirmation',JSON.stringify(customerConfirmation));
    customerBasket=[];
    saveCustomerBasket();
    renderOrderConfirmation();
    window.location.hash='customer-confirmation';
  }catch(orderError){
    error.textContent=orderError.message==='Failed to fetch'?'We couldn’t place your order. Please check your connection and try again.':orderError.message.startsWith('A product')||orderError.message.startsWith('A basket')?orderError.message:'We couldn’t place your order. Please try again.';
    error.hidden=false;
  }finally{
    checkoutSubmitting=false;
    button.disabled=false;
    button.innerHTML='Place order <span aria-hidden="true">→</span>';
  }
}
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
function refundRequestStatusLabel(value){return ({requested:'Requested',approved:'Approved',rejected:'Rejected',refunded:'Refunded'})[String(value||'').toLowerCase()]||'Unknown'}
function renderAdminRefundRequests(){
  const count=adminRefundRequests.filter(request=>String(request.status||'').toLowerCase()==='requested').length;
  $('#refundRequestCount').textContent=count;
  $('#refundRequestCount').hidden=count===0;
  $('#refundRequestKpis').innerHTML=[['Awaiting review',count],['All requests',adminRefundRequests.length]].map(item=>'<div class="mini-card"><label>'+item[0]+'</label><strong>'+item[1]+'</strong></div>').join('');
  const body=$('#refundRequestsTable');
  if(adminRefundRequestsLoading){body.innerHTML='<tr><td colspan="8">Loading refund requests…</td></tr>';return}
  if(adminRefundRequestsError){body.innerHTML='<tr><td colspan="8">Refund requests are unavailable. Use Refresh to try again.</td></tr>';return}
  if(!adminRefundRequests.length){body.innerHTML='<tr><td colspan="8">No refund requests found.</td></tr>';return}
  body.innerHTML=adminRefundRequests.map(request=>{
    const id=String(request.id),requestStatus=String(request.status||'').toLowerCase(),busy=adminRefundActions.has(id);
    return '<tr><td><b>#'+escapeHtml(id)+'</b></td><td>#'+escapeHtml(request.order_id)+'</td><td>'+escapeHtml(request.customer_id)+'</td><td>'+money(request.amount,request.currency||'GBP')+'</td><td class="refund-reason-cell">'+escapeHtml(request.reason||'—')+'</td><td><span class="refund-status-badge '+escapeHtml(requestStatus)+'">'+escapeHtml(refundRequestStatusLabel(requestStatus))+'</span></td><td>'+escapeHtml(formatCustomerOrderDate(request.created_at))+'</td><td>'+(requestStatus==='requested'?'<div class="refund-admin-actions"><button class="primary-btn" type="button" data-refund-action="approve" data-refund-id="'+escapeHtml(id)+'" '+(busy?'disabled':'')+'>Approve</button><button class="secondary-btn" type="button" data-refund-action="reject" data-refund-id="'+escapeHtml(id)+'" '+(busy?'disabled':'')+'>Reject</button></div>':'—')+'</td></tr>';
  }).join('');
}
async function loadAdminRefundRequests(){
  if(adminRefundRequestsLoading)return;
  adminRefundRequestsLoading=true;
  adminRefundRequestsError=false;
  renderAdminRefundRequests();
  try{
    const response=await api('/api/payments/refund-requests');
    if(!Array.isArray(response))throw new Error('Invalid refund requests response');
    adminRefundRequests=response;
  }catch(error){
    adminRefundRequestsError=true;
    console.error('Could not load refund requests',error);
  }finally{
    adminRefundRequestsLoading=false;
    renderAdminRefundRequests();
  }
}
async function handleAdminRefundAction(button){
  const id=button.dataset.refundId,action=button.dataset.refundAction;
  if(!id||adminRefundActions.has(id))return;
  const word=action==='approve'?'approve':'reject';
  if(!window.confirm('Are you sure you want to '+word+' refund request #'+id+'?'))return;
  adminRefundActions.add(id);
  renderAdminRefundRequests();
  try{
    await api('/api/payments/refund-requests/'+encodeURIComponent(id)+'/'+word,{method:'POST'});
    await loadAdminRefundRequests();
  }catch(error){
    console.error('Could not '+word+' refund request',error);
    await loadAdminRefundRequests();
    toast('Could not '+word+' this refund request. Please refresh and try again.');
  }finally{
    adminRefundActions.delete(id);
    renderAdminRefundRequests();
  }
}
function renderServices(){const s=[['dashboard-api','Public UI & analytics','8086'],['api-gateway','Internal routing & auth','8080'],['order-service','Order lifecycle','8081'],['inventory-service','Stock & reservations','8082'],['payment-service','Payments & ledger','8083'],['notification-service','Email & SMS','8084'],['shipping-service','Shipping & tracking','8085'],['worker','SQS orchestration','—'],['scheduler','Scheduled jobs','—']];$('#serviceGrid').innerHTML=s.map((x,i)=>'<article class="service-card"><div class="service-top"><span class="service-icon">'+(i+1)+'</span><span class="service-state">● Configured</span></div><h3>'+x[0]+'</h3><p>'+x[1]+'</p><div class="service-meta"><div><span>Port</span><b>'+x[2]+'</b></div></div></article>').join('')}
function renderOrderCounts(){$('#ordersNavCount').textContent=state.orders.length;$$('#orderTabs button').forEach(b=>{const n=b.dataset.status==='all'?state.orders.length:state.orders.filter(o=>o.status===b.dataset.status).length;const counters=Array.from(b.querySelectorAll('*')).filter(e=>/^\D*\d+\D*$/.test(e.textContent.trim()));if(counters.length)counters[counters.length-1].textContent=counters[counters.length-1].textContent.replace(/\d+/,n);else{const text=Array.from(b.childNodes).find(node=>node.nodeType===Node.TEXT_NODE&&/\d+/.test(node.textContent));if(text)text.textContent=text.textContent.replace(/\d+/,n)}})}
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
  const view = window.location.hash.replace('#', '');
  const dashboardViews = ['overview','orders','inventory','shipments','payments','refunds','services'];
  const customerViews=['customer-shop','customer-basket','customer-checkout','customer-confirmation','customer-orders'];
  const isAdmin=sessionStorage.getItem('bravo_token')&&sessionStorage.getItem(ADMIN_UI_SESSION)==='authenticated';
  const isCustomer=sessionStorage.getItem('bravo_token')&&sessionStorage.getItem(CUSTOMER_UI_SESSION)==='authenticated';

  if (view === 'admin-login') {
    if(isCustomer){window.location.hash='customer-shop';return}
    $('#landingPage').hidden = true;
    $('#loginPage').hidden = false;
    $('#customerLoginPage').hidden = true;
    $('#customerPortal').hidden = true;
    $('#appShell').hidden = true;
    return;
  }

  if (view === 'customer-login') {
    if(isAdmin){window.location.hash='overview';return}
    if(isCustomer){window.location.hash='customer-shop';return}
    $('#landingPage').hidden = true;
    $('#loginPage').hidden = true;
    $('#customerLoginPage').hidden = false;
    $('#customerPortal').hidden = true;
    $('#appShell').hidden = true;
    return;
  }

  if (dashboardViews.includes(view)) {
    if(isCustomer){window.location.hash='customer-shop';return}
    if (!isAdmin) {
      sessionStorage.removeItem('bravo_token');
      sessionStorage.removeItem(ADMIN_UI_SESSION);
      sessionStorage.removeItem(CUSTOMER_UI_SESSION);
      $('#landingPage').hidden = false;
      $('#loginPage').hidden = true;
      $('#customerLoginPage').hidden = true;
      $('#customerPortal').hidden = true;
      $('#appShell').hidden = true;
      window.location.hash = 'home';
      return;
    }

    $('#landingPage').hidden = true;
    $('#loginPage').hidden = true;
    $('#customerLoginPage').hidden = true;
    $('#customerPortal').hidden = true;
    $('#appShell').hidden = false;
    switchView(view, false);
    if (!dashboardStarted) {
      dashboardStarted = true;
      renderAll();
      loadLive();
      loadAdminRefundRequests();
    }else if(view==='orders'){
      loadLive();
    }
    if(view==='refunds')loadAdminRefundRequests();
    return;
  }

  if(customerViews.includes(view)){
    if(isAdmin){window.location.hash='overview';return}
    if(!isCustomer){
      sessionStorage.removeItem('bravo_token');
      sessionStorage.removeItem(CUSTOMER_UI_SESSION);
      sessionStorage.removeItem(ADMIN_UI_SESSION);
      $('#landingPage').hidden=false;
      $('#loginPage').hidden=true;
      $('#customerLoginPage').hidden=true;
      $('#customerPortal').hidden=true;
      $('#appShell').hidden=true;
      window.location.hash='home';
      return;
    }
    $('#landingPage').hidden=true;
    $('#loginPage').hidden=true;
    $('#customerLoginPage').hidden=true;
    $('#customerPortal').hidden=false;
    $('#appShell').hidden=true;
    $('#customerShopView').hidden=view!=='customer-shop';
    $('#customerBasketView').hidden=view!=='customer-basket';
    $('#customerCheckoutView').hidden=view!=='customer-checkout';
    $('#customerConfirmationView').hidden=view!=='customer-confirmation';
    $('#customerOrdersView').hidden=view!=='customer-orders';
    $('#basketUpdateMessage').hidden=true;
    $$('.customer-nav-link').forEach(button=>button.classList.toggle('active',button.dataset.customerView===(view==='customer-orders'?'orders':'shop')));
    renderCustomerBasket();
    if(view==='customer-checkout'){
      if(!customerBasket.length){window.location.hash='customer-basket';return}
      restoreCheckoutDetails();
      renderCheckout();
    }
    if(view==='customer-confirmation')renderOrderConfirmation();
    if(view==='customer-shop'||view==='customer-basket'||view==='customer-checkout')loadCustomerProducts();
    if(view==='customer-orders')loadCustomerOrders();
    return;
  }

  if(isCustomer){window.location.hash='customer-shop';return}
  $('#landingPage').hidden = false;
  $('#loginPage').hidden = true;
  $('#customerLoginPage').hidden = true;
  $('#customerPortal').hidden = true;
  $('#appShell').hidden = true;
}

window.addEventListener('hashchange', loadViewFromHash);

function toast(m){const t=$('#toast');t.textContent=m;t.classList.add('show');setTimeout(()=>t.classList.remove('show'),2200)}
$('#customerLoginForm').onsubmit=async e=>{
  e.preventDefault();
  const error=$('#customerLoginError'),submit=$('#customerLoginForm button[type="submit"]');
  const email=$('#customerLoginEmail').value,password=$('#customerLoginPassword').value;
  error.hidden=true;
  if(email==='customer@bravoorders.com'&&password==='customer123'){
    submit.disabled=true;
    sessionStorage.removeItem('bravo_token');
    sessionStorage.removeItem(ADMIN_UI_SESSION);
    sessionStorage.removeItem(CUSTOMER_UI_SESSION);
    sessionStorage.removeItem('bravo_redirect');
    try{
      const response=await fetch('/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email,password})});
      if(!response.ok)throw new Error('Sign-in failed. Please try again.');
      const result=await response.json();
      if(!result.token)throw new Error('Sign-in response did not include an authentication token.');
      sessionStorage.setItem('bravo_token',result.token);
      sessionStorage.setItem(CUSTOMER_UI_SESSION,'authenticated');
      customerProducts=[];
      customerProductsLoaded=false;
      customerProductsError='';
      window.location.hash='customer-shop';
    }catch(loginError){
      error.textContent=loginError instanceof TypeError?'Unable to reach Bravo Orders. Check your connection and try again.':loginError.message;
      error.hidden=false;
    }finally{
      submit.disabled=false;
    }
    return;
  }
  error.textContent='Invalid email or password.';
  error.hidden=false;
};
$('#loginForm').onsubmit=async e=>{
  e.preventDefault();
  const error=$('#loginError'),submit=$('#loginSubmit');
  error.hidden=true;
  submit.disabled=true;
  sessionStorage.removeItem('bravo_token');
  sessionStorage.removeItem(ADMIN_UI_SESSION);
  sessionStorage.removeItem(CUSTOMER_UI_SESSION);
  const email=$('#loginEmail').value,password=$('#loginPassword').value;
  if(email!=='admin@bravoorders.com'||password!=='admin123'){
    error.textContent='Invalid email or password.';
    error.hidden=false;
    submit.disabled=false;
    return;
  }
  try{
    const response=await fetch('/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email,password})});
    if(!response.ok)throw new Error(response.status===401||response.status===403?'Invalid email or password.':'Sign-in failed. Please try again.');
    const result=await response.json();
    if(!result.token)throw new Error('Sign-in response did not include an authentication token.');
    sessionStorage.setItem('bravo_token',result.token);
    sessionStorage.setItem(ADMIN_UI_SESSION,'authenticated');
    const redirect=sessionStorage.getItem('bravo_redirect');
    sessionStorage.removeItem('bravo_redirect');
    window.location.hash=['overview','orders','inventory','shipments','payments','services'].includes(redirect)?redirect:'overview';
  }catch(loginError){
    error.textContent=loginError instanceof TypeError?'Unable to reach Bravo Orders. Check your connection and try again.':loginError.message;
    error.hidden=false;
  }finally{
    submit.disabled=false;
  }
};
$('#signOutBtn').onclick=()=>{
  sessionStorage.removeItem('bravo_token');
  sessionStorage.removeItem(ADMIN_UI_SESSION);
  sessionStorage.removeItem(CUSTOMER_UI_SESSION);
  sessionStorage.removeItem('bravo_redirect');
  state={orders:[],products:[],shipments:[],ledger:[],source:'loading'};
  dashboardStarted=false;
  window.location.hash='home';
};
$('#customerLogoutBtn').onclick=()=>{
  sessionStorage.removeItem('bravo_token');
  sessionStorage.removeItem(CUSTOMER_UI_SESSION);
  sessionStorage.removeItem(ADMIN_UI_SESSION);
  sessionStorage.removeItem(CUSTOMER_BASKET);
  sessionStorage.removeItem('bravo_checkout_details');
  sessionStorage.removeItem('bravo_customer_confirmation');
  sessionStorage.removeItem('bravo_customer_order_summaries');
  customerBasket=[];
  customerProducts=[];
  customerProductsLoaded=false;
  customerProductsError='';
  customerConfirmation=null;
  customerOrderSummaries={};
  customerOrders=[];
  selectedCustomerOrderId=null;
  window.location.hash='home';
};
function navigateCustomerView(view){
  const next=view==='basket'?'customer-basket':view==='orders'?'customer-orders':'customer-shop';
  if(next===window.location.hash.replace('#','')){if(next==='customer-orders')loadCustomerOrders();return}
  window.location.hash=next;
}
$('#customerPortal').onclick=e=>{
  const button=e.target.closest('[data-customer-view]');
  if(button)navigateCustomerView(button.dataset.customerView);
};
$('#basketControl').onclick=()=>{window.location.hash='customer-basket'};
$('#checkoutBtn').onclick=()=>{
  $('#checkoutError').hidden=true;
  window.location.hash='customer-checkout';
};
$('#checkoutForm').onsubmit=placeCustomerOrder;
$$('#checkoutForm input').forEach(input=>input.oninput=saveCheckoutDetails);
$('#customerProductGrid').onclick=e=>{
  const retry=e.target.closest('#retryProductsBtn');
  if(retry){loadCustomerProducts(true);return}
  const add=e.target.closest('[data-add-product]');
  if(add)addCustomerProduct(add.dataset.addProduct);
};
$('#customerBasketItems').onclick=e=>{
  const action=e.target.closest('[data-basket-action]');
  if(action)changeCustomerBasket(action.dataset.productId,action.dataset.basketAction);
};
$('#customerOrderList').onclick=e=>{
  if(e.target.closest('#retryCustomerOrdersBtn')){loadCustomerOrders();return}
  const card=e.target.closest('[data-customer-order-id]');
  if(!card)return;
  selectedCustomerOrderId=card.dataset.customerOrderId;
  renderCustomerOrders();
};
$('#refreshCustomerOrdersBtn').onclick=()=>loadCustomerOrders();
$('#refreshProductsBtn').onclick=()=>loadCustomerProducts(true);
$('#refundRequestsRefreshBtn').onclick=()=>loadAdminRefundRequests();
$('#refundRequestsTable').onclick=event=>{
  const button=event.target.closest('[data-refund-action]');
  if(button)handleAdminRefundAction(button);
};
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