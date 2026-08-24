package console

import (
	"fmt"
	"net/http"
)

const pageShell = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><title>%s</title>
<style>
body{font-family:system-ui,sans-serif;margin:2rem;background:#f6f7f9;color:#222}
h1{font-size:1.3rem} table{border-collapse:collapse;background:#fff;width:100%%}
th,td{border:1px solid #ddd;padding:.45rem .6rem;text-align:left;font-size:.9rem}
th{background:#eef1f5} .badge{display:inline-block;padding:.1rem .5rem;border-radius:999px;font-size:.75rem}
.live{background:#d9f6dd;color:#0a7a2f}.failed{background:#fde2e2;color:#b00020}
.draft{background:#e8ecf1;color:#555}
</style></head><body>
<nav><a href="/">总览</a> · <a href="/console/tasks">任务</a> · <a href="/console/streams">流</a> · <a href="/console/nodes">节点</a> · <a href="/console/quality">质量</a></nav>
<h1>%s</h1>
<div id="app">加载中…</div>
<script>%s</script>
</body></html>`

const commonJS = `
async function getJSON(url){const r=await fetch(url);return r.json()}
function esc(v){return String(v).replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]))}
function table(rows,cols){return '<table><tr>'+cols.map(c=>'<th>'+c+'</th>').join('')+'</tr>'+rows.map(r=>'<tr>'+r.map(c=>'<td>'+c+'</td>').join('')+'</tr>').join('')+'</table>'}
function stateBadge(s){const m={'live':'live','failed':'failed','draft':'draft'};return '<span class="badge '+(m[s]||'draft')+'">'+esc(s)+'</span>'}
`

func (a *API) index(w http.ResponseWriter, _ *http.Request) {
	script := commonJS + `
Promise.all([getJSON('/api/streams'),getJSON('/api/jobs')]).then(([s,j])=>{
  const live=s.filter(x=>x.state==='live').length;
  const jobs=j.filter(x=>x.state==='done').length;
  document.getElementById('app').innerHTML =
    '<p>共 '+s.length+' 条流，'+live+' 条已上线；任务 '+j.length+' 个，完成 '+jobs+' 个。</p>'+
    table(s.map(x=>[esc(x.title),stateBadge(x.state),x.segments,x.revision,x.success.toFixed(2)]),['标题','状态','分片','版本','通过率']);
})`
	servePage(w, "EdgeTranscode 总览", "EdgeTranscode 总览", script)
}

func (a *API) tasksPage(w http.ResponseWriter, _ *http.Request) {
	script := commonJS + `
getJSON('/api/jobs').then(j=>{
  document.getElementById('app').innerHTML = table(
    j.map(x=>[esc(x.id.slice(0,8)),esc(x.stream_id.slice(0,8)),esc(x.profile),stateBadge(x.state),x.attempts,esc(x.error||'')]),
    ['任务','流','档位','状态','尝试','错误']);
})`
	servePage(w, "转码任务", "转码任务", script)
}

func (a *API) streamsPage(w http.ResponseWriter, _ *http.Request) {
	script := commonJS + `
getJSON('/api/streams').then(s=>{
  document.getElementById('app').innerHTML = table(
    s.map(x=>[esc(x.title),esc(x.owner),stateBadge(x.state),x.segments,x.revision,(x.success*100).toFixed(1)+'%']),
    ['标题','作者','状态','分片','版本','QC 通过率']);
})`
	servePage(w, "内容流", "内容流", script)
}

func (a *API) nodesPage(w http.ResponseWriter, _ *http.Request) {
	script := commonJS + `
getJSON('/api/nodes').then(n=>{
  document.getElementById('app').innerHTML = table(
    n.map(x=>[esc(x.name),esc(x.region),x.segments,x.cache_entries,x.cdn_objects]),
    ['节点','区域','分片数','缓存条目','CDN 对象']);
})`
	servePage(w, "边缘节点", "边缘节点", script)
}

func (a *API) qualityPage(w http.ResponseWriter, _ *http.Request) {
	script := commonJS + `
getJSON('/api/quality').then(q=>{
  document.getElementById('app').innerHTML = table(
    q.slice().reverse().map(x=>[esc(x.stream_id.slice(0,8)),esc(x.segment_id),x.score.toFixed(2),x.passed?'通过':'失败',esc(x.at)]),
    ['流','分片','分数','结果','时间']);
})`
	servePage(w, "质量采样", "质量采样", script)
}

func servePage(w http.ResponseWriter, title, heading, script string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fmt.Sprintf(pageShell, title, heading, script)))
}
