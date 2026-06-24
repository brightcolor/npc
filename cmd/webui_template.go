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
.form-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:1rem}.table-actions{display:flex;gap:.35rem;flex-wrap:wrap}.mini-form{display:inline}.section-note{max-width:880px}
</style>
</head>
<body class="hold-transition sidebar-mini layout-fixed dark-mode">
<div class="wrapper">
<nav class="main-header navbar navbar-expand navbar-dark">
<ul class="navbar-nav"><li class="nav-item"><a class="nav-link" data-widget="pushmenu" href="#"><i class="fas fa-bars"></i></a></li><li class="nav-item d-none d-sm-inline-block"><span class="nav-link">Reverse proxy control plane</span></li></ul>
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
{{if .Notice}}<div class="alert alert-{{.Notice.Kind}}"><strong>{{.Notice.Title}}</strong><br>{{.Notice.Message}}</div>{{end}}
<div class="row"><div class="col-lg-8"><div class="card"><div class="card-header"><h3 class="card-title"><i class="fas fa-plus-circle mr-2"></i>Create site</h3></div><div class="card-body">
<form method="post" action="/actions"><input type="hidden" name="action" value="create"><div class="form-grid">
<div class="form-group"><label>Hostname</label><input name="hostname" class="form-control" placeholder="app.example.com" required></div>
<div class="form-group"><label>Backend host</label><input name="backend_host" class="form-control" value="127.0.0.1" required></div>
<div class="form-group"><label>Backend port</label><input name="backend_port" type="number" min="1" max="65535" class="form-control" value="3000" required></div>
<div class="form-group"><label>Backend scheme</label><select name="backend_scheme" class="form-control"><option>http</option><option>https</option></select></div>
<div class="form-group"><label>Profile</label><select name="profile" class="form-control"><option>generic</option><option>websocket</option><option>upload</option><option>streaming</option><option>docker</option><option>security-basic</option></select></div>
<div class="form-group"><label>Body size</label><input name="client_max_body_size" class="form-control" value="100M"></div>
</div><div class="form-grid">
<label><input type="checkbox" name="ssl"> SSL</label><label><input type="checkbox" name="acme"> acme.sh</label><label><input type="checkbox" name="redirect_https" checked> HTTPS redirect</label><label><input type="checkbox" name="websocket"> WebSocket</label><label><input type="checkbox" name="http2" checked> HTTP/2</label><label><input type="checkbox" name="access_log" checked> Access log</label><label><input type="checkbox" name="error_log" checked> Error log</label><label><input type="checkbox" name="force"> Replace existing</label>
</div><div class="form-grid mt-3"><div class="form-group"><label>Certificate path</label><input name="cert_path" class="form-control" placeholder="/path/fullchain.pem"></div><div class="form-group"><label>Key path</label><input name="key_path" class="form-control" placeholder="/path/privkey.pem"></div><div class="form-group"><label>ACME method</label><select name="acme_method" class="form-control"><option value="http">http</option><option value="dns">dns</option><option value="standalone">standalone</option></select></div><div class="form-group"><label>DNS provider</label><input name="dns_provider" class="form-control" placeholder="cloudflare"></div></div>
<label class="mt-2"><input type="checkbox" name="confirm" value="yes" required> Apply this site creation</label><br><button class="btn btn-info mt-2"><i class="fas fa-save mr-1"></i>Create site</button></form>
</div></div></div>
<div class="col-lg-4"><div class="card card-outline card-info"><div class="card-header"><h3 class="card-title"><i class="fas fa-file-import mr-2"></i>Import existing configs</h3></div><div class="card-body">
<p class="text-muted section-note">Adopt manually created or third-party Nginx reverse proxy configs from sites-available.</p><form method="post" action="/actions"><input type="hidden" name="action" value="import"><div class="form-group"><label>Config path, optional</label><input name="path" class="form-control" placeholder="/etc/nginx/sites-available/app.conf"></div><label><input type="checkbox" name="force"> Refresh existing metadata</label><br><label><input type="checkbox" name="confirm" value="yes" required> Import configs</label><br><button class="btn btn-outline-info mt-2">Import</button></form>
</div></div></div></div>
<div class="row"><div class="col-lg-6"><div class="card"><div class="card-header"><h3 class="card-title"><i class="fas fa-edit mr-2"></i>Edit site</h3></div><div class="card-body"><form id="editForm" method="post" action="/actions"><input type="hidden" name="action" value="edit"><div class="form-grid"><div class="form-group"><label>Site</label><select name="hostname" class="form-control site-picker">{{range .Sites}}<option data-host="{{.BackendHost}}" data-port="{{.BackendPort}}" data-scheme="{{.BackendScheme}}" data-profile="{{.Profile}}" data-body="{{.ClientMaxBodySize}}" data-websocket="{{.WebSocket}}" data-redirect="{{.RedirectHTTPS}}" data-http2="{{.HTTP2}}" data-cert="{{.CertificatePath}}" data-key="{{.CertificateKeyPath}}" data-acme="{{.ACME}}" data-method="{{.ACMEMethod}}" data-provider="{{.DNSProvider}}">{{.Hostname}}</option>{{end}}</select></div><div class="form-group"><label>Backend host</label><input name="backend_host" class="form-control" placeholder="127.0.0.1"></div><div class="form-group"><label>Backend port</label><input name="backend_port" type="number" min="1" max="65535" class="form-control"></div><div class="form-group"><label>Backend scheme</label><select name="backend_scheme" class="form-control"><option>http</option><option>https</option></select></div><div class="form-group"><label>Profile</label><input name="profile" class="form-control" placeholder="generic"></div><div class="form-group"><label>Body size</label><input name="client_max_body_size" class="form-control" placeholder="100M"></div></div><label><input type="checkbox" name="websocket"> WebSocket</label> <label><input type="checkbox" name="redirect_https"> HTTPS redirect</label> <label><input type="checkbox" name="http2"> HTTP/2</label> <label><input type="checkbox" name="no_reload"> Skip reload</label><br><label class="mt-2"><input type="checkbox" name="confirm" value="yes" required> Apply edit</label><br><button class="btn btn-info mt-2">Save changes</button></form></div></div></div>
<div class="col-lg-6"><div class="card"><div class="card-header"><h3 class="card-title"><i class="fas fa-certificate mr-2"></i>Certificates</h3></div><div class="card-body"><form id="certSetForm" method="post" action="/actions"><input type="hidden" name="action" value="cert-set"><div class="form-grid"><div class="form-group"><label>Site</label><select name="hostname" class="form-control site-picker">{{range .Sites}}<option data-cert="{{.CertificatePath}}" data-key="{{.CertificateKeyPath}}" data-acme="{{.ACME}}" data-method="{{.ACMEMethod}}" data-provider="{{.DNSProvider}}" data-redirect="{{.RedirectHTTPS}}" data-http2="{{.HTTP2}}">{{.Hostname}}</option>{{end}}</select></div><div class="form-group"><label>Fullchain path</label><input name="cert_path" class="form-control" required></div><div class="form-group"><label>Key path</label><input name="key_path" class="form-control" required></div><div class="form-group"><label>DNS provider</label><input name="dns_provider" class="form-control"></div></div><label><input type="checkbox" name="manual" checked> Manual certificate</label> <label><input type="checkbox" name="acme"> acme.sh managed</label> <label><input type="checkbox" name="redirect_https" checked> HTTPS redirect</label> <label><input type="checkbox" name="http2" checked> HTTP/2</label><br><label class="mt-2"><input type="checkbox" name="confirm" value="yes" required> Update certificate</label><br><button class="btn btn-info mt-2">Set certificate</button></form><hr><form method="post" action="/actions"><input type="hidden" name="action" value="cert-issue"><div class="form-grid"><div class="form-group"><label>Site</label><select name="hostname" class="form-control">{{range .Sites}}<option>{{.Hostname}}</option>{{end}}</select></div><div class="form-group"><label>Method</label><select name="method" class="form-control"><option value="http">HTTP-01</option><option value="dns">DNS-01</option></select></div><div class="form-group"><label>DNS provider</label><input name="dns_provider" class="form-control" placeholder="cloudflare"></div><div class="form-group"><label>Email, optional</label><input name="email" class="form-control"></div></div><input type="hidden" name="acme_ca" value="letsencrypt"><label><input type="checkbox" name="redirect_https" checked> HTTPS redirect</label> <label><input type="checkbox" name="http2" checked> HTTP/2</label> <label><input type="checkbox" name="no_reload"> Skip reload</label><br><label class="mt-2"><input type="checkbox" name="confirm" value="yes" required> Issue certificate now</label><br><button class="btn btn-success mt-2">Issue certificate</button></form><hr><form method="post" action="/actions"><input type="hidden" name="action" value="cert-delete"><div class="form-group"><label>Site</label><select name="hostname" class="form-control">{{range .Sites}}<option>{{.Hostname}}</option>{{end}}</select></div><label><input type="checkbox" name="keep_acme"> Keep acme.sh registration</label> <label><input type="checkbox" name="remove_files"> Delete cert files</label> <label><input type="checkbox" name="no_reload"> Skip reload</label><br><label class="mt-2"><input type="checkbox" name="confirm" value="yes" required> Remove certificate from site</label><br><button class="btn btn-danger mt-2">Delete certificate</button></form></div></div></div></div>
<div class="row"><div class="col-12"><div class="card"><div class="card-header"><h3 class="card-title"><i class="fas fa-globe mr-2"></i>Managed sites</h3></div><div class="card-body p-0 table-wrap"><table class="table table-hover table-striped mb-0"><thead><tr><th>Hostname</th><th>State</th><th>SSL</th><th>Backend</th><th>Profile</th><th>Group</th><th>Tags</th><th>Actions</th><th>Config</th></tr></thead><tbody>
{{range .Sites}}<tr><td><strong>{{.Hostname}}</strong></td><td><span class="badge badge-{{if eq .State "on"}}success{{else}}secondary{{end}}">{{.State}}</span></td><td><span class="badge badge-{{if eq .SSL "yes"}}success{{else}}warning{{end}}">{{.SSL}}</span></td><td><code>{{.Backend}}</code></td><td>{{.Profile}}</td><td>{{.Group}}</td><td>{{.Tags}}</td><td><div class="table-actions"><form class="mini-form" method="post" action="/actions"><input type="hidden" name="action" value="enable"><input type="hidden" name="hostname" value="{{.Hostname}}"><input type="hidden" name="confirm" value="yes"><button class="btn btn-xs btn-success">Enable</button></form><form class="mini-form" method="post" action="/actions"><input type="hidden" name="action" value="disable"><input type="hidden" name="hostname" value="{{.Hostname}}"><input type="hidden" name="confirm" value="yes"><button class="btn btn-xs btn-secondary">Disable</button></form><form class="mini-form" method="post" action="/actions"><input type="hidden" name="action" value="delete"><input type="hidden" name="hostname" value="{{.Hostname}}"><input type="hidden" name="remove_metadata" value="on"><input type="hidden" name="confirm" value="yes"><button class="btn btn-xs btn-danger" onclick="return confirm('Delete {{.Hostname}} metadata?')">Delete</button></form></div></td><td class="path" title="{{.Config}}">{{.Config}}</td></tr>{{else}}<tr><td colspan="9" class="text-muted">No npc-managed sites found.</td></tr>{{end}}
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
function checked(form,name,value){const el=form.querySelector('[name="'+name+'"]'); if(el) el.checked=value==='true'}
function fillForm(form){
  const select=form.querySelector('.site-picker'); if(!select) return;
  const o=select.options[select.selectedIndex]; if(!o) return;
  const set=(name,value)=>{const el=form.querySelector('[name="'+name+'"]'); if(el && value!==undefined) el.value=value};
  set('backend_host',o.dataset.host); set('backend_port',o.dataset.port); set('backend_scheme',o.dataset.scheme);
  set('profile',o.dataset.profile); set('client_max_body_size',o.dataset.body);
  set('cert_path',o.dataset.cert); set('key_path',o.dataset.key); set('dns_provider',o.dataset.provider);
  checked(form,'websocket',o.dataset.websocket); checked(form,'redirect_https',o.dataset.redirect); checked(form,'http2',o.dataset.http2); checked(form,'acme',o.dataset.acme);
}
document.querySelectorAll('form').forEach(form=>{fillForm(form); const s=form.querySelector('.site-picker'); if(s) s.addEventListener('change',()=>fillForm(form));});
</script>
</body>
</html>`
