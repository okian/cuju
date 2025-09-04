async function load(){
  const res = await fetch('./search-index.json');
  const idx = await res.json();
  const sb = document.getElementById('sidebar');
  sb.innerHTML = '<input id="search" placeholder="Search docs..." />\n<div class="nav" id="nav"></div>';
  const nav = document.getElementById('nav');
  const render = (items)=>{
    nav.innerHTML = items.map(i=>`<a href="${i.path}">${i.title}</a>`).join('');
  };
  render(idx);
  const input = document.getElementById('search');
  input.addEventListener('input', ()=>{
    const q = input.value.toLowerCase();
    if(!q){ render(idx); return; }
    const f = idx.filter(i=>i.title.toLowerCase().includes(q)||i.text.toLowerCase().includes(q));
    render(f);
	});
}
window.addEventListener('DOMContentLoaded', load);
