/* Offline-first recorded-event explorer. No external map service is contacted. */
(() => {
  const byID = id => document.getElementById(id);
  const readSaved = (key, fallback) => { try { return JSON.parse(localStorage.getItem(key)) || fallback; } catch (_) { return fallback; } };
  let data = {rows:[],days:[],locations:{},modes:[]}, page=0, selected='', requestID=0, detailRequest=0;
  let heatRows=[], mapRows=[], bookmarks=readSaved('gpsdr-explore-bookmarks',{}), views=readSaved('gpsdr-explore-views',[]);
  const filterIDs=['query','from','to','min','max','location','mode'];
  const labels = row => row.location ? (row.location.label || `${row.location.latitude.toFixed(2)}, ${row.location.longitude.toFixed(2)}`) : 'No location';
  const e = escapeHTML;
  const filteredRows = () => {
    const rows = data.rows.filter(row=>!byID('explore-bookmarks-only').checked || bookmarks[row.key]);
    const sort=byID('explore-sort').value;
    return rows.sort((a,b)=>sort==='frequency'?a.frequencyHz-b.frequencyHz:sort==='recent'?new Date(b.last)-new Date(a.last):sort==='decoded'?b.decoded-a.decoded:b.count-a.count);
  };
  function queryParams() { const query=new URLSearchParams();for(const id of filterIDs){const value=byID('explore-'+id).value;if(value)query.set(id==='query'?'q':id,value);}return query; }
  function refreshChoices() {
    for(const [id,values,title] of [['location',Object.entries(data.locations),'All locations'],['mode',data.modes.map(m=>[m,m]),'All modes']]) {
      const select=byID('explore-'+id),value=select.value;
      select.innerHTML=`<option value="">${title}</option>`+values.sort((a,b)=>a[1].localeCompare(b[1])).map(([key,label])=>`<option value="${e(key)}">${e(label)}</option>`).join('');select.value=value;
    }
  }
  async function refresh() {
    const id=++requestID;byID('explore-status').textContent='Loading local history…';
    try {
      const result=await api('/api/explore?'+queryParams()); if(id!==requestID)return;
      data=result;page=0;refreshChoices();
      if(selected&&!data.rows.some(row=>row.key===selected)){selected='';detailRequest++;byID('explore-detail').textContent='Select a frequency to inspect its captures.';}
      render();
      byID('explore-status').textContent=`${data.events.toLocaleString()} matching recorded events from ${data.loadedEvents.toLocaleString()} loaded journal records (up to 25,000). Simulated events excluded. Audio/IQ counts are journal links, not a file-integrity check.`;
    } catch(error) { if(id===requestID)byID('explore-status').textContent='Could not load history: '+error.message; }
  }
  function render() {
    byID('explore-stats').innerHTML=[[data.events,'Recorded events'],[data.rows.length,'Frequency / area groups'],[compactDuration(data.seconds),'Sum of capture durations'],[data.locatedEvents,'Events with location']].map(([value,label])=>`<div><strong>${e(String(value))}</strong><span>${label}</span></div>`).join('');
    const rows=filteredRows(),pages=Math.max(1,Math.ceil(rows.length/100));page=Math.min(page,pages-1);
    byID('explore-page').textContent=`${page+1} / ${pages}`;byID('explore-prev').disabled=page===0;byID('explore-next').disabled=page+1>=pages;
    byID('explore-rows').innerHTML=rows.slice(page*100,(page+1)*100).map(row=>`<tr class="${row.key===selected?'explore-selected':''}"><td><button data-explore-bookmark="${e(row.key)}" aria-label="Bookmark ${e(formatFrequency(row.frequencyHz))}" aria-pressed="${!!bookmarks[row.key]}">${bookmarks[row.key]?'★':'☆'}</button></td><td><button class="explore-row-button" data-explore-key="${e(row.key)}">${e(formatFrequency(row.frequencyHz))}</button>${row.label?`<small>${e(row.label)}</small>`:''}</td><td>${e(labels(row))}${bookmarks[row.key]?.tag?`<small>${e(bookmarks[row.key].tag)}</small>`:''}</td><td>${row.count}</td><td>${e(compactDuration(row.seconds))}</td><td>${e(row.modes.join(', '))}</td><td>${row.decoded} / ${row.transcribed}</td></tr>`).join('')||'<tr><td colspan="7">No observations match these filters.</td></tr>';
    drawHeat(rows);drawTimeline();drawMap(rows);
  }
  function drawHeat(rows) {
    heatRows=rows.slice(0,60);const canvas=byID('explore-heatmap');canvas.height=Math.max(95,heatRows.length*23+35);
    const ctx=canvas.getContext('2d'),left=180,cell=(1000-left)/24;
    ctx.fillStyle='#080c12';ctx.fillRect(0,0,1000,canvas.height);ctx.font='11px monospace';ctx.textBaseline='middle';
    for(let h=0;h<24;h++){ctx.fillStyle='#9aa8b8';ctx.fillText(String(h).padStart(2,'0'),left+h*cell+4,15);}
    const maximum=Math.max(1,...heatRows.flatMap(row=>row.hourly));
    heatRows.forEach((row,index)=>{const y=30+index*23;ctx.fillStyle='#c5cdd7';ctx.fillText(`${(row.frequencyHz/1e6).toFixed(4)} · ${row.location?'GPS':'—'}`,5,y+9);row.hourly.forEach((count,h)=>{const strength=Math.log1p(count)/Math.log1p(maximum);ctx.fillStyle=count?`hsl(${190-strength*35} 60% ${18+strength*44}%)`:'#111821';ctx.fillRect(left+h*cell,y,cell-2,21);});});
    if(!heatRows.length){ctx.fillStyle='#98a5b7';ctx.fillText('No recorded activity',15,55);}
  }
  function drawTimeline() {
    const canvas=byID('explore-timeline'),ctx=canvas.getContext('2d'),days=data.days.slice(-500),max=Math.max(1,...days.map(d=>d.count));
    ctx.fillStyle='#080c12';ctx.fillRect(0,0,700,220);ctx.font='11px monospace';ctx.fillStyle='#aebdce';ctx.fillText(`${max} events (peak) · latest ${days.length} observed days`,15,20);
    if(!days.length)return;
    const first=Date.parse(days[0].day),last=Date.parse(days.at(-1).day),span=Math.max(86400000,last-first+86400000),width=Math.max(.6,Math.min(25,660*86400000/span-1));
    days.forEach(d=>{const x=20+660*(Date.parse(d.day)-first)/span,height=d.count/max*155;ctx.fillStyle='#70b8cf';ctx.fillRect(x,190-height,width,height);});
    ctx.fillStyle='#aebdce';ctx.fillText(days[0].day,20,210);ctx.fillText(days.at(-1).day,590,210);
  }
  function drawMap(rows) {
    mapRows=rows.filter(row=>row.location);const svg=byID('explore-map'),refs=mapRows.flatMap(row=>row.references||[]);
    const coords=[...mapRows.map(row=>row.location),...refs.map(ref=>ref.area)];
    if(!coords.length){svg.innerHTML='<text x="30" y="130" fill="#aebdce" font-size="14">No coordinates recorded for this selection.</text>';return;}
    let minLat=Math.min(...coords.map(c=>c.latitude)),maxLat=Math.max(...coords.map(c=>c.latitude)),minLon=Math.min(...coords.map(c=>c.longitude)),maxLon=Math.max(...coords.map(c=>c.longitude));
    const padLat=Math.max(.02,(maxLat-minLat)*.15),padLon=Math.max(.02,(maxLon-minLon)*.15);minLat-=padLat;maxLat+=padLat;minLon-=padLon;maxLon+=padLon;
    const xy=c=>[55+(c.longitude-minLon)/(maxLon-minLon)*610,240-(c.latitude-minLat)/(maxLat-minLat)*215];
    let content='';for(let i=0;i<=4;i++){const x=55+i*610/4,y=25+i*215/4;content+=`<path d="M${x},25V240 M55,${y}H665" stroke="#253341" fill="none"/><text x="${x}" y="263" text-anchor="middle" fill="#98a7ba" font-size="10">${(minLon+i*(maxLon-minLon)/4).toFixed(2)}°</text><text x="4" y="${y+4}" fill="#98a7ba" font-size="10">${(maxLat-i*(maxLat-minLat)/4).toFixed(2)}°</text>`;}
    const uniqueRefs=new Map(refs.map(ref=>[`${ref.area.latitude},${ref.area.longitude},${ref.area.label}`,ref]));for(const ref of uniqueRefs.values()){const [x,y]=xy(ref.area);content+=`<path d="M${x},${y-6}l6,6 -6,6 -6,-6Z" fill="#bf9cea"><title>${e(ref.provider+' reference area: '+ref.area.label+'; not a transmitter site')}</title></path>`;}
    // Aggregate point markers; exact coincident groups remain selectable in the table.
    const points=new Map();for(const row of mapRows){const key=`${row.location.latitude},${row.location.longitude}`;const p=points.get(key)||{location:row.location,count:0,key:row.key};p.count+=row.count;points.set(key,p);}
    for(const point of points.values()){const [x,y]=xy(point.location);content+=`<circle cx="${x}" cy="${y}" r="${Math.min(14,4+Math.log1p(point.count))}" fill="#69b9d0" fill-opacity=".7" stroke="#bdeaff" tabindex="0" role="button" data-explore-key="${e(point.key)}" aria-label="Inspect collection point"><title>${e(point.location.label||'Collection point')} · ${point.count} events. Additional frequencies at this location are listed below.</title></circle>`;}
    svg.innerHTML=content;
  }
  async function selectRow(key) {
    const row=data.rows.find(r=>r.key===key);if(!row)return;selected=key;render();const id=++detailRequest;
    const detail=byID('explore-detail');detail.innerHTML=`<h2>${e(formatFrequency(row.frequencyHz))} · ${e(labels(row))}</h2><p>${row.count} events · ${row.audio} audio links · ${row.iq} IQ links · peak ${row.peakDBFS.toFixed(1)} dBFS<br>${e(row.first)} → ${e(row.last)}<br>Callsigns: ${e(row.callsigns.join(', ')||'None decoded or transcribed')}</p><div class="explore-tag"><label for="explore-tag-text">Bookmark tag</label><input id="explore-tag-text" maxlength="120" value="${e(bookmarks[key]?.tag||'')}" placeholder="Local repeater, investigate…"><button id="explore-tag-save" type="button">Save</button></div><h3>Nearby reference candidates</h3>${row.references.length?row.references.map(ref=>`<p class="explore-reference">${e(ref.name)} · ${e(ref.provider)}<br>${e(ref.area.label)} · ${ref.distanceMiles.toFixed(1)} mi from reference area center · reference radius ${ref.area.radiusMiles} mi<br><small>Imported ${e(ref.area.importedAt||'date unavailable')}. Frequency and area match only; not proof of transmitter identity or location.</small></p>`).join(''):'<p>No geographically applicable imported reference for this group. Untagged recordings cannot receive a location-based match.</p>'}<h3>Latest captures in this group · all dates · up to 50</h3><div id="explore-captures">Loading…</div>`;
    byID('explore-tag-save').onclick=()=>{bookmarks[key]={tag:byID('explore-tag-text').value.trim()};localStorage.setItem('gpsdr-explore-bookmarks',JSON.stringify(bookmarks));render();toast('Bookmark saved in this app/browser');};
    try{const captures=await api('/api/explore/captures?key='+encodeURIComponent(key));if(id!==detailRequest)return;byID('explore-captures').innerHTML=captures.map(event=>`<details><summary>${e(new Date(event.startedAt).toLocaleString())} · ${e(event.modulation||'Unknown')} · ${Number(event.durationSeconds).toFixed(2)} s · ${e(event.analysisStatus||'Not analyzed')}</summary>${event.recovered?'<p>Recovered file · original identity and reception metadata unavailable</p>':''}${(event.mediaIssues||[]).map(issue=>`<p>${e(issue)}</p>`).join('')}<p>${e(event.capturePolicy==='archive'?'Original IQ archive · shared receiver capture':event.captureID?'Filtered channel capture':'Legacy capture')}</p><p>${e(event.transcript||'No speech transcript')}</p><p>${e(event.analysis?.summary||'No model summary')}</p><pre>${e(JSON.stringify(event.decoderMessages||[],null,2))}</pre>${event.audioPath?`<audio controls preload="none" src="/api/audio?id=${encodeURIComponent(event.id)}&token=${encodeURIComponent(serverToken)}"></audio>`:''}</details>`).join('')||'No journal captures available.';}catch(error){if(id===detailRequest)byID('explore-captures').textContent=error.message;}
  }
  byID('explore-filters').addEventListener('submit',event=>{event.preventDefault();refresh();});
  byID('explore-reset').onclick=()=>{for(const id of filterIDs)byID('explore-'+id).value='';refresh();};
  for(const id of ['sort','bookmarks-only'])byID('explore-'+id).onchange=()=>{page=0;render();};
  byID('explore-prev').onclick=()=>{page--;render();};byID('explore-next').onclick=()=>{page++;render();};
  byID('view-explore').addEventListener('click',event=>{const bookmark=event.target.closest('[data-explore-bookmark]');if(bookmark){const key=bookmark.dataset.exploreBookmark;if(bookmarks[key])delete bookmarks[key];else bookmarks[key]={tag:''};localStorage.setItem('gpsdr-explore-bookmarks',JSON.stringify(bookmarks));render();return;}const item=event.target.closest('[data-explore-key]');if(item)selectRow(item.dataset.exploreKey);});
  byID('explore-map').addEventListener('keydown',event=>{if(event.key==='Enter'&&event.target.dataset.exploreKey)selectRow(event.target.dataset.exploreKey);});
  byID('explore-heatmap').onclick=event=>{const canvas=event.currentTarget,rect=canvas.getBoundingClientRect(),y=(event.clientY-rect.top)*canvas.height/rect.height,index=Math.floor((y-30)/23);if(heatRows[index])selectRow(heatRows[index].key);};
  byID('explore-heatmap').onmousemove=event=>{const canvas=event.currentTarget,rect=canvas.getBoundingClientRect(),x=(event.clientX-rect.left)*1000/rect.width,y=(event.clientY-rect.top)*canvas.height/rect.height,row=heatRows[Math.floor((y-30)/23)],hour=Math.floor((x-180)/((1000-180)/24));canvas.title=row&&hour>=0&&hour<24?`${formatFrequency(row.frequencyHz)} · ${labels(row)} · ${hour}:00 UTC · ${row.hourly[hour]} recorded events`:'';};
  function savedChoices(){byID('explore-saved').innerHTML='<option value="">Choose…</option>'+views.map((v,i)=>`<option value="${i}">${e(v.name)}</option>`).join('');}
  byID('explore-save-view').onclick=()=>{if(views.length>=30)return toast('Remove a saved view first (limit 30)',true);views.push({name:`View ${views.length+1} · ${byID('explore-query').value||byID('explore-location').selectedOptions[0]?.textContent||'All activity'}`,filters:filterIDs.map(id=>byID('explore-'+id).value)});localStorage.setItem('gpsdr-explore-views',JSON.stringify(views));savedChoices();toast('View saved in this app/browser');};
  byID('explore-saved').onchange=event=>{const saved=views[Number(event.target.value)];if(event.target.value===''||!saved)return;filterIDs.forEach((id,index)=>byID('explore-'+id).value=saved.filters[index]);refresh();};
  byID('explore-delete-view').onclick=()=>{const value=byID('explore-saved').value;if(value==='')return;views.splice(Number(value),1);localStorage.setItem('gpsdr-explore-views',JSON.stringify(views));savedChoices();};
  document.querySelector('[data-view="explore"]').addEventListener('click',refresh);
  savedChoices();if(state.view==='explore')refresh();
})();
