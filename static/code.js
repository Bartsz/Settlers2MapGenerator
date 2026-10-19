
const $=id=>document.getElementById(id),names=['Coal','Iron','Gold','Granite (Mine)','Surface Stone'],colors=['#80b44e','#347c32','#9aca5c','#b29a68','#787c83','#267cba'];let clusters=[],variants=[],current=0,last=null,working=false,previewParamsKey='';

const resourceRows=[
  ['Coal',150,3,1000,40], ['Iron',50,2,1000,40], ['Gold',20,1,1000,40],
  ['Granite (Mine)',40,2,1000,40], ['Surface Stone',30,2,1000,40],
  ['Forests',100,4,1000,40], ['Lakes (tiles/lake)',25,3,120,10],
  ['Wildlife',40,3,1000,40], ['Empty Mountain Terrain',0,0,1000,40]
];
$('resources').innerHTML=resourceRows.map(([name,count,groups,maxCount,maxGroups],i)=>
 `<div class="resource-row"><label for="t${i}">${name}</label><div class="quantity"><input id="t${i}" type="range" min="${i===6?1:0}" max="${maxCount}" value="${count}" oninput="$('tv${i}').textContent=this.value"><output id="tv${i}">${count}</output></div><input id="g${i}" title="Number of Clusters" aria-label="${name} Clusters" type="number" min="0" max="${maxGroups}" value="${groups}"></div>`).join('');


function syncSymmetryLimits(){
 const exact=[3,5,6].includes(Number($('sym').value));
 const slider=$('diameter');slider.min=exact?65:64;slider.max=exact?139:140;slider.step=2;
 document.querySelectorAll('[data-sym]').forEach(b=>{b.classList.toggle('selected',b.dataset.sym===$('sym').value);b.setAttribute('aria-pressed',String(b.dataset.sym===$('sym').value))});
 $('roundStartSettings').hidden=!exact;
 $('projectionMode').disabled=!exact;
 $('projectionMode').value='raw';
 const oldSize=Number(slider.value);
 slider.value=Math.max(Number(slider.min),Math.min(Number(slider.max),oldSize+(oldSize%2===(exact?1:0)?0:1)));
 updateDiameterLabel();
}
function selectSymmetry(n){$('sym').value=String(n);syncSymmetryLimits();refresh(true)}
function updateDiameterLabel(){const d=Number($('diameter').value),rot=[3,5,6].includes(Number($('sym').value));$('diameterValue').textContent=d;$('diameterHint').textContent=rot?'Map '+d+'×'+d+' (odd dimensions). Circular playable area with mountains outside.':'Map '+d+'×'+d+' (even dimensions). Fully usable square; mirrored sectors.';}
syncSymmetryLimits();
function params(){return {seed:Number($('seed').value)||0,sym:Number($('sym').value),water:13,lakes:Number($('g6').value),lakeSize:Number($('t6').value),mountain:23,forest:Number($('forest').value),diameter:Number($('diameter').value),landLevel:Number($('landLevel').value),startRadiusPercent:Number($('startRadiusPercent').value),miningTerrainExpansion:Number($('miningTerrainExpansion').value),totals:names.map((_,i)=>Number($('t'+i).value)),groups:names.map((_,i)=>Number($('g'+i).value)),forestTiles:Number($('t5').value),forestGroups:Number($('g5').value),surfaceStoneGroups:Number($('g4').value),animals:Number($('t7').value),animalGroups:Number($('g7').value),emptyMountainTiles:Number($('t8').value),emptyMountainGroups:Number($('g8').value),variant:variants[current]?.map(a=>a.count)||[],variantTypes:variants[current]?.map(a=>a.type)||[]}}
async function api(endpoint,p){const r=await fetch(endpoint,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(p)});if(!r.ok)throw Error('Server '+r.status+': '+(await r.text()).slice(0,350));return r}
function showGenerationError(message){
 const el=$('generationError');
 el.textContent=message;
 el.hidden=false;
}
function clearGenerationError(){
 const el=$('generationError');
 el.textContent='';
 el.hidden=true;
}
async function refresh(reset=false){if(working)return;working=true;$('status').textContent='Generating...';try{let p=params();if(reset){p.variant=[];p.variantTypes=[]}let m=await (await api('/api/map',p)).json();if(reset||!variants.length||clusters.length!==m.clusters.length){clusters=m.clusters;variants=[clusters.map(c=>({type:c.type,count:(c.count??c.capacity)}))];current=0;m=await(await api('/api/map',params())).json()}else{clusters=clusters.map((c,i)=>({...c}))}last=m;previewParamsKey=JSON.stringify(params());draw(m);drawTable();$('status').textContent='Map ready.'+(m.issues?.length?' Warnings: '+m.issues.join('; '):'');clearGenerationError();return true}catch(e){$('status').textContent='';showGenerationError('Error: '+e.message);return false}finally{working=false}}
const depositColors=['#000000','#272727','#eb6434','#ffd34a','#b77aff','#f4f4e8','#25d9ff','#ff75ba'];
const depositNames=['','Coal','Iron','Gold','Granite (Mine)','Surface Stones','Fish','Wildlife'];
$('legend').innerHTML=depositNames.slice(1).map((n,i)=>`<span class="legend-item"><i class="swatch" style="background:${depositColors[i+1]}"></i>${n}</span>`).join('') + '<span class="legend-item"><i class="swatch" style="background:#184a23"></i>Forest with Wildlife</span>' + '<span class="legend-item"><i class="swatch" style="background:transparent;border-style:dashed"></i>No Deposit</span>';
// Circular map generation, sectors and SWD coordinates use the same raw X/Y
// geometry. Hex projection is an OPTIONAL inspection mode; it may distort
// an actually round X/Y map and does not affect generation or export.
const HEX_Y=Math.sqrt(3)/2, MAP_PIXELS=720;let mapUnits=109,mapScale=720/109,mapCenter=54;
function axialToScreen(x,y,hex){
  if(!hex) return [(x+.5)*mapScale,(y+.5)*mapScale];
  const q=x-mapCenter,r=y-mapCenter;
  return [MAP_PIXELS/2+mapScale*(q+r/2),MAP_PIXELS/2+mapScale*HEX_Y*r];
}
function cubeRound(q,r){
  // Correct nearest axial hex; independently rounding q and r would produce
  // seams and off-by-one resource marks along the projected sector borders.
  const s=-q-r;
  let qi=Math.round(q),ri=Math.round(r),si=Math.round(s);
  const dq=Math.abs(q-qi),dr=Math.abs(r-ri),ds=Math.abs(s-si);
  if(dq>dr&&dq>ds)qi=-ri-si;
  else if(dr>ds)ri=-qi-si;
  return [qi,ri];
}
function screenToMap(px,py,hex){
  if(!hex)return [Math.floor(px/mapScale),Math.floor(py/mapScale)];
  const u=(px-MAP_PIXELS/2)/mapScale;
  const v=(py-MAP_PIXELS/2)/mapScale;
  const r=v/HEX_Y;
  const [q0,r0]=cubeRound(u-r/2,r);
  return [mapCenter+q0,mapCenter+r0];
}
function useHexProjection(){return $('projectionMode').value==='hex';}
function draw(m){
 mapUnits=m.width;mapScale=MAP_PIXELS/mapUnits;mapCenter=(mapUnits-1)/2;
 const ctx=$('map').getContext('2d');
 const hex=useHexProjection();
 const pixels=ctx.createImageData(MAP_PIXELS,MAP_PIXELS);
 const rgb=[[128,180,78],[52,124,50],[154,202,92],[24,74,35],[120,124,131],[38,124,186],[166,143,102]];
 const view=$('viewMode').value, overlay=m.deposits||[];
 const cols=depositColors.map(c=>[parseInt(c.slice(1,3),16),parseInt(c.slice(3,5),16),parseInt(c.slice(5,7),16)]);
 // Native-quality raster resampling makes projected resources match SWD cells.
 // Rendering 720x720 keeps individual tiles legible without smoothing.
 for(let py=0;py<MAP_PIXELS;py++){
  for(let px=0;px<MAP_PIXELS;px++){
   const [x,y]=screenToMap(px+.5,py+.5,hex);
   const inside=x>=0&&x<mapUnits&&y>=0&&y<mapUnits;
   const i=inside?y*mapUnits+x:-1;
   const dep=inside?(overlay[i]||0):0;
   let v=inside?(rgb[m.tiles[i]]||rgb[4]):rgb[4];
   if(view==='resources')v=dep?cols[dep]:[36,51,59];
   else if(view==='both'&&dep&&!(m.tiles[i]===3&&dep===7))v=cols[dep];
   const off=(py*MAP_PIXELS+px)*4;
   pixels.data[off]=v[0];pixels.data[off+1]=v[1];pixels.data[off+2]=v[2];pixels.data[off+3]=255;
  }
 }
 ctx.putImageData(pixels,0,0);
 // Symmetry lines denote SECTOR BOUNDARIES, not headquarters axes.
 ctx.strokeStyle='#eff8fa';ctx.lineWidth=2;ctx.setLineDash([9,7]);
 const sym=Number($('sym').value);
 if(sym===2||sym===4){
   ctx.beginPath();ctx.moveTo(360,0);ctx.lineTo(360,720);ctx.stroke();
   if(sym===4){ctx.beginPath();ctx.moveTo(0,360);ctx.lineTo(720,360);ctx.stroke();}
 }else{
   // Compute every overlay point in the exact X/Y system used by generation.
   // Optional hex preview transforms these points, not the generated map.
   const c=(mapUnits-1)/2, radius=c;
   const project=(x,y)=>axialToScreen(x,y,hex);
   const [cx,cy]=project(c,c);
   for(let i=0;i<sym;i++){
     const a=(i+.5)*Math.PI*2/sym;
     const [ex,ey]=project(c+(radius+1)*Math.cos(a),c+(radius+1)*Math.sin(a));
     ctx.beginPath();ctx.moveTo(cx,cy);ctx.lineTo(ex,ey);ctx.stroke();
   }
   ctx.strokeStyle='#fff8cb';ctx.lineWidth=2;ctx.setLineDash([5,8]);ctx.beginPath();
   for(let j=0;j<=180;j++){
     const a=j*2*Math.PI/180;
     const [px,py]=project(c+radius*Math.cos(a),c+radius*Math.sin(a));
     if(j===0)ctx.moveTo(px,py);else ctx.lineTo(px,py);
   }
   ctx.closePath();ctx.stroke();
 }
 ctx.setLineDash([]);
 (m.harbors||[]).forEach(h=>{
   const [x,y]=axialToScreen(h[0],h[1],hex);
   ctx.beginPath();ctx.arc(x,y,8,0,Math.PI*2);ctx.fillStyle='#54e2b9';ctx.fill();ctx.strokeStyle='#062d30';ctx.lineWidth=2;ctx.stroke();ctx.fillStyle='#062d30';ctx.textAlign='center';ctx.textBaseline='middle';ctx.font='bold 10px Segoe UI';ctx.fillText('P',x,y);
 });
 (m.hq||[]).forEach((h,i)=>{
   const [x,y]=axialToScreen(h[0],h[1],hex);
   ctx.beginPath();ctx.arc(x,y,12,0,Math.PI*2);ctx.fillStyle='#fff0b8';ctx.fill();ctx.fillStyle='#263027';ctx.textAlign='center';ctx.textBaseline='middle';ctx.font='bold 15px Segoe UI';ctx.fillText(i+1,x,y);
 });
 $('stats').textContent='Variant: '+(current+1)+' / '+variants.length+' · Mining clusters: '+m.clusters.length+' · Source sector: '+((m.symmetryReport?.templateSourceSector??0)+1)+' · Size: '+m.width+'×'+m.height+' · Area: '+(sym===2||sym===4?'square':'circle + outer mountains')+' · Surface stone: '+(m.clusters||[]).filter(c=>c.type===4).length+' clusters · Lakes: '+(m.lakesReport?.perPlayer??'—')+' × '+(m.lakesReport?.tilesPerLake??'—')+' tiles · Height: '+$('landLevel').value+' · Projection: '+(useHexProjection()?'hexagonal':'raw grid');
 const info=m.symmetryReport;
 if(info){
    const per=(key)=>info[key]||[];
    const series=['lakes','water','coal','iron','gold','granite','rocks','fish','animals'];
    const same=series.every(k=>per(k).length===info.players && per(k).every(v=>Number.isFinite(v)&&v===per(k)[0]));
    $('balance').innerHTML='<p><strong>'+(same?([3,5,6].includes(info.players)?'EQUAL RESOURCE COUNTS (RASTERIZED ROTATION)':'EQUAL RESOURCE COUNTS AND MIRRORING'):'WARNING: SECTOR IMBALANCE')+'</strong> — '+info.playablePerPlayer+' playable tiles per player (excluding the outer mountain belt). Data read from the final SWD.</p>'+
       '<table><thead><tr><th>Player</th><th>Lakes</th><th>Water</th><th>Coal</th><th>Iron</th><th>Gold</th><th>Granite</th><th>Stone</th><th>Fish</th><th>Wildlife</th></tr></thead><tbody>'+Array.from({length:info.players},(_,i)=>'<tr><td>'+(i+1)+'</td>'+series.map(k=>'<td>'+(per(k)[i]??'—')+'</td>').join('')+'</tr>').join('')+'</tbody></table>';
 }else $('balance').textContent='No symmetry report.';
}

function openDrawer(){$('drawer').classList.add('show');$('shade').classList.add('show');drawTable()}function closeDrawer(){$('drawer').classList.remove('show');$('shade').classList.remove('show')}
function drawTable(){if(!clusters.length)return;const escapeHTML=s=>String(s).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;');let h='<table><thead><tr><th>Variant</th>'+clusters.map((c,i)=>`<th>Cluster ${i+1}<div class="tiny">${names[c.type]}<br>max ${c.capacity}</div></th>`).join('')+'</tr></thead><tbody>';variants.forEach((variant,r)=>{h+=`<tr class="${r===current?'active':''}"><td><label><input type="radio" name="variant" ${r===current?'checked':''} onchange="pick(${r})"> ${r+1}</label></td>`+variant.map((entry,i)=>`<td><select aria-label="Cluster type" onchange="setType(${r},${i},this.value)">${names.map((n,k)=>`<option value="${k}" ${entry.type===k?'selected':''} ${((k===4)!=(clusters[i].type===4))?'disabled':''}>${escapeHTML(n)}</option>`).join('')}</select><div><input aria-label="Tile count" type="number" min="0" max="${clusters[i].capacity}" value="${entry.count}" onchange="setAmount(${r},${i},this.value)"> / ${clusters[i].capacity}</div></td>`).join('')+'</tr>'});$('variantTable').innerHTML=h+'</tbody></table>';}
function pick(i){current=i;applyVariant()}function setType(r,i,v){variants[r][i].type=Number(v);drawTable();if(r===current)refresh()}function setAmount(r,i,v){variants[r][i].count=Math.max(0,Math.min(clusters[i].capacity,Number(v)||0));drawTable();if(r===current)refresh()}function addVariant(){if(!variants.length)return;variants.push(variants[current].map(v=>({...v})));current=variants.length-1;drawTable()}function applyVariant(){refresh();drawTable()}
function newMap(){$('seed').value=Math.floor(Math.random()*2147483647);refresh(true)}
async function downloadSWD(){try{if(working)throw Error('Generation is still in progress');if(!last||previewParamsKey!==JSON.stringify(params())){const ok=await refresh();if(!ok)throw Error('Could not update preview — export cancelled')}let response=await api('/api/download',params());let blob=await response.blob();saveBlob(blob,'Settlers_'+$('seed').value+'_'+$('sym').value+'p_'+$('diameter').value+'x'+$('diameter').value+'_D'+$('diameter').value+'_H'+$('landLevel').value+'.swd');$('status').textContent='SWD saved from the displayed preview.'}catch(e){showGenerationError('Export error: '+e.message)}}
function saveBlob(blob,name){let url=URL.createObjectURL(blob);let a=document.createElement('a');a.href=url;a.download=name;a.click();setTimeout(()=>URL.revokeObjectURL(url),3000)}
function saveProject(){saveBlob(new Blob([JSON.stringify({version:3,settings:params(),clusters,variants,current},null,2)],{type:'application/json'}),'Settlers_project_'+$('seed').value+'.json')}
async function loadProject(e){try{const obj=JSON.parse(await e.target.files[0].text());if(obj.version!==2&&obj.version!==3)throw Error('Unsupported project version');let p=obj.settings;for(let k of ['seed','sym','forest'])$(k).value=p[k];$('diameter').value=p.diameter||([3,5,6].includes(Number(p.sym))?109:108);$('landLevel').value=([1,40,100].includes(Number(p.landLevel))?Number(p.landLevel):1);$('startRadiusPercent').value=p.startRadiusPercent??55;$('startRadiusValue').textContent=$('startRadiusPercent').value;$('miningTerrainExpansion').value=p.miningTerrainExpansion??3;$('miningTerrainExpansionValue').textContent=$('miningTerrainExpansion').value;syncSymmetryLimits();for(let i=0;i<5;i++){$('t'+i).value=p.totals?.[i]??[150,50,20,40,30][i];if(i<4)$('g'+i).value=p.groups?.[i]??[3,2,1,2,2][i];$('tv'+i).textContent=$('t'+i).value}$('g4').value=Math.max(0,Math.min(40,Number(p.surfaceStoneGroups??p.groups?.[4]??2)));$('forestv').textContent=p.forest??27;$('t7').value=p.animals??40;$('tv7').textContent=$('t7').value;$('g7').value=p.animalGroups??3;
$('t8').value=p.emptyMountainTiles??0;$('tv8').textContent=$('t8').value;$('g8').value=p.emptyMountainGroups??0;$('g6').value=p.lakes??3;$('t6').value=p.lakeSize??25;$('tv6').textContent=$('t6').value;$('t5').value=p.forestTiles??100;$('tv5').textContent=$('t5').value;$('g5').value=p.forestGroups??4;clusters=obj.clusters;variants=obj.variants;current=obj.current;await refresh()}catch(err){showGenerationError('Could not open project: '+err.message)}}
refresh(true);

async function saveAndRun(){
 try {
   if(working) throw Error('Generation is still in progress');
   if(!last||previewParamsKey!==JSON.stringify(params())) { const ok=await refresh();if(!ok)throw Error('Cannot save an invalid map'); }
   const r=await api('/api/save-and-run',params());const result=await r.json();
   $('status').textContent='Saved map: '+result.path+'; requested opening in the associated application.';
 } catch(e) {showGenerationError('Save and Run: '+e.message);}
}
