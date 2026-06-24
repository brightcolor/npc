package cmd

const webUIHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>npc web UI</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@5.15.4/css/all.min.css">
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/admin-lte@3.2/dist/css/adminlte.min.css">
<style>
.brand-icon{height:38px;width:38px;border-radius:10px;display:grid;place-items:center;font-weight:800}
.content-wrapper{background:#f4f6f9}.dark-mode .content-wrapper{background:#101418}
.small-box,.card{border-radius:.6rem}.card{box-shadow:0 10px 30px rgba(0,0,0,.08)}
.dark-mode .card,.dark-mode .small-box{box-shadow:0 12px 34px rgba(0,0,0,.28)}
.table td,.table th{vertical-align:middle;white-space:nowrap}.table-wrap{overflow-x:auto}
.path{max-width:320px;overflow:hidden;text-overflow:ellipsis}.hint{max-width:820px}.main-footer{font-size:.875rem}
.polish-gradient{background:linear-gradient(135deg,#1976d2,#00bcd4);color:#fff}
.dark-mode .polish-gradient{background:linear-gradient(135deg,#0d47a1,#00838f)}
.command-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:.5rem}.cmd-card{border:1px solid rgba(127,127,127,.2);border-radius:.45rem;padding:.65rem;background:rgba(127,127,127,.06);cursor:pointer}
.cmd-card:hover{border-color:#17a2b8}.console{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;min-height:130px;white-space:pre-wrap}
</style>
</head>
<body class="hold-transition sidebar-mini layout-fixed dark-mode">
<div class="wrapper">
<nav class="main-header navbar navbar-expand navbar-dark">
<ul class="navbar-nav"><li class="nav-item"><a class="nav-link" data-widget="pushmenu" href="#"><i class="fas fa-bars"></i></a></li><li class="nav-item d-none d-sm-inline-block"><span class="nav-link">Read-only control plane</span></li></ul>
<ul class="navbar-nav ml-auto"><li class="nav-item"><button id="themeToggle" class="btn btn-sm btn-outline-info"><i class="fas fa-adjust"></i> Dark mode</button></li></ul>
</nav>
<aside class="main-sidebar sidebar-dark-primary elevation-4">
<a href="/" class="brand-link"><span class="brand-icon polish-gradient brand-image elevation-2">npc</span><span class="brand-text font-weight-light">Proxy Configurator</span></a>
<div class="sidebar"><nav class="mt-3"><ul class="nav nav-pills nav-sidebar flex-column"><li class="nav-item"><a class="nav-link active" href="/"><i class="nav-icon fas fa-server"></i><p>Dashboard</p></a></li><li class="nav-item"><a class="nav-link" href="/api/sites"><i class="nav-icon fas fa-code"></i><p>Sites API</p></a></li><li class="nav-item"><a class="nav-link" href="/api/status"><i class="nav-icon fas fa-heartbeat"></i><p>Status API</p></a></li></ul></nav></div>
</aside>
<div class="content-wrapper">
<section class="content-header"><div class="container-fluid"><div class="row mb-2"><div class="col-sm-8"><h1>Nginx Proxy Dashboard</h1><p class="text-muted mb-0">AdminLTE web view for npc-managed reverse proxies.</p></div><div class="col-sm-4 text-sm-right"><span class="badge badge-info">npc {{.Version}}</span></div></div></div></section>
<section class="content"><div class="container-fluid">
<div class="row">
<div class="col-lg-3 col-6"><div class="small-box polish-gradient"><div class="inner"><h3>{{index .Status "active_sites"}}</h3><p>Active sites</p></div><div class="icon"><i class="fas fa-toggle-on"></i></div></div></div>
<div class="col-lg-3 col-6"><div class="small-box bg-secondary"><div class="inner"><h3>{{index .Status "disabled_sites"}}</h3><p>Disabled sites</p></div><div class="icon"><i class="fas fa-toggle-off"></i></div></div></div>
<div class="col-lg-3 col-6"><div class="small-box bg-success"><div class="inner"><h3>{{index .Status "nginx_active"}}</h3><p>Nginx active</p></div><div class="icon"><i class="fas fa-heart"></i></div></div></div>
<div class="col-lg-3 col-6"><div class="small-box bg-info"><div class="inner"><h3>{{index .Status "matched_sites"}}</h3><p>Total managed</p></div><div class="icon"><i class="fas fa-layer-group"></i></div></div></div>
</div>
<div class="row"><div class="col-lg-8"><div class="card"><div class="card-header"><h3 class="card-title"><i class="fas fa-terminal mr-2"></i>Operations</h3></div><div class="card-body">
<form method="post" action="/run"><div class="form-group"><label for="command">npc command</label><div class="input-group"><div class="input-group-prepend"><span class="input-group-text">npc</span></div><input id="command" name="command" class="form-control" value="{{.Command}}" placeholder="list --wide"></div><small class="form-text text-muted">Commands run through the npc binary without a shell. Write actions require confirmation below.</small></div>
<div class="form-group form-check"><input class="form-check-input" type="checkbox" name="confirm" value="yes" id="confirm"><label class="form-check-label" for="confirm">I understand this may change Nginx, certificates, backups, services, or npc metadata.</label></div><button class="btn btn-info"><i class="fas fa-play mr-1"></i>Run command</button></form>
{{if .RunResult}}<hr><h5>Result: <span class="badge badge-{{if .RunResult.OK}}success{{else}}danger{{end}}">{{if .RunResult.OK}}ok{{else}}failed{{end}}</span></h5><p class="text-muted mb-1">{{.RunResult.Command}}</p>{{if .RunResult.Error}}<div class="alert alert-danger">{{.RunResult.Error}}</div>{{end}}<pre class="console bg-dark text-light p-3 rounded">{{.RunResult.Output}}</pre>{{end}}
<hr><div class="command-grid">{{range .Catalog}}<div class="cmd-card" data-command="{{.Command}}"><strong>{{.Title}}</strong>{{if .Risky}} <span class="badge badge-warning">write</span>{{end}}<br><code>{{.Command}}</code></div>{{end}}</div>
</div></div></div>
<div class="col-lg-4"><div class="card card-outline card-info"><div class="card-header"><h3 class="card-title"><i class="fas fa-shield-alt mr-2"></i>Security model</h3></div><div class="card-body"><p class="hint">The web UI can run npc commands. Put it behind your reverse proxy authentication, bind it to a private interface when possible, and use HTTPS at the proxy.</p><dl class="row"><dt class="col-5">Listen mode</dt><dd class="col-7">configured by <code>--listen</code></dd><dt class="col-5">Shell</dt><dd class="col-7">not used</dd><dt class="col-5">Write guard</dt><dd class="col-7">confirmation checkbox</dd></dl></div></div></div></div>
<div class="row"><div class="col-12"><div class="card"><div class="card-header"><h3 class="card-title"><i class="fas fa-globe mr-2"></i>Managed sites</h3></div><div class="card-body p-0 table-wrap"><table class="table table-hover table-striped mb-0"><thead><tr><th>Hostname</th><th>State</th><th>SSL</th><th>Backend</th><th>Profile</th><th>Group</th><th>Tags</th><th>Config</th></tr></thead><tbody>
{{range .Sites}}<tr><td><strong>{{.Hostname}}</strong></td><td><span class="badge badge-{{if eq .State "on"}}success{{else}}secondary{{end}}">{{.State}}</span></td><td><span class="badge badge-{{if eq .SSL "yes"}}success{{else}}warning{{end}}">{{.SSL}}</span></td><td><code>{{.Backend}}</code></td><td>{{.Profile}}</td><td>{{.Group}}</td><td>{{.Tags}}</td><td class="path" title="{{.Config}}">{{.Config}}</td></tr>{{else}}<tr><td colspan="8" class="text-muted">No npc-managed sites found.</td></tr>{{end}}
</tbody></table></div></div></div>
<div class="row"><div class="col-12"><div class="card"><div class="card-header"><h3 class="card-title"><i class="fas fa-info-circle mr-2"></i>Runtime</h3></div><div class="card-body"><dl class="row mb-0"><dt class="col-md-2">Config</dt><dd class="col-md-10 text-truncate">{{index .Status "config_file"}}</dd><dt class="col-md-2">Sites available</dt><dd class="col-md-10 text-truncate">{{index .Status "sites_available"}}</dd><dt class="col-md-2">Nginx version</dt><dd class="col-md-10 text-truncate">{{index .Status "nginx_version"}}</dd></dl></div></div></div></div>
</div></section></div>
<footer class="main-footer"><strong>npc web UI</strong><span class="float-right d-none d-sm-inline">AdminLTE, dark mode, reverse-proxy-auth ready.</span></footer>
</div>
<script src="https://cdn.jsdelivr.net/npm/jquery@3.6.4/dist/jquery.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/bootstrap@4.6.2/dist/js/bootstrap.bundle.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/admin-lte@3.2/dist/js/adminlte.min.js"></script>
<script>
const key='npc-webui-theme';
const saved=localStorage.getItem(key);
if(saved==='light'){document.body.classList.remove('dark-mode')}
document.getElementById('themeToggle').addEventListener('click',()=>{document.body.classList.toggle('dark-mode');localStorage.setItem(key,document.body.classList.contains('dark-mode')?'dark':'light')});
document.querySelectorAll('.cmd-card').forEach(card=>card.addEventListener('click',()=>{document.getElementById('command').value=card.dataset.command;window.scrollTo({top:0,behavior:'smooth'})}));
</script>
</body>
</html>`
