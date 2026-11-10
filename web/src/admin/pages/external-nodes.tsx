import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { api, errorText, unwrap } from "../../api/client";
import { Button, EmptyState, PageHeader, Pill, Skeleton } from "../../components/ui";
import { useToast } from "../../components/toast";

type ExternalNode = { id:number; name:string; protocol:string; enabled:boolean; auto:boolean; server:string; port:number };
const key=["external-nodes"] as const;

export function ExternalNodesPage(){
 const qc=useQueryClient(); const toast=useToast();
 const [json,setJSON]=useState(""); const [name,setName]=useState("");
 const list=useQuery({queryKey:key,queryFn:async()=>(await unwrap(api.GET("/api/v1/external-nodes" as never) as never)) as unknown as ExternalNode[]});
 const refresh=()=>void qc.invalidateQueries({queryKey:key});
 const add=useMutation({mutationFn:async()=>unwrap(api.POST("/api/v1/external-nodes/import" as never,{body:{json,name:name.trim()||undefined}} as never) as never),onSuccess:()=>{setJSON("");setName("");refresh();toast.ok("JSON-серверы импортированы")},onError:e=>toast.error(errorText(e))});
 const patch=useMutation({mutationFn:async(v:{id:number;body:Record<string,unknown>})=>unwrap(api.PATCH("/api/v1/external-nodes/{id}" as never,{params:{path:{id:v.id}},body:v.body} as never) as never),onSettled:refresh,onError:e=>toast.error(errorText(e))});
 const remove=useMutation({mutationFn:async(id:number)=>unwrap(api.DELETE("/api/v1/external-nodes/{id}" as never,{params:{path:{id}}} as never) as never),onSettled:refresh,onError:e=>toast.error(errorText(e))});
 return <>
  <PageHeader title="Внешние ноды" sub="Импорт Xray JSON в обычную подписку Mikan и группу Auto" />
  <section className="card p-4 space-y-3">
   <h2 className="font-semibold">Импорт JSON</h2>
   <label className="field"><span>Собственное название (необязательно)</span><input className="input" value={name} onChange={e=>setName(e.target.value)} placeholder="🇦🇪 UAE Premium" maxLength={120}/></label>
   <label className="field"><span>Полный Xray JSON</span><textarea className="input mono min-h-[220px]" value={json} onChange={e=>setJSON(e.target.value)} placeholder={'{\n  "outbounds": [...]\n}'} spellCheck={false}/></label>
   <p className="text-sm muted">Импортируются поддерживаемые клиентские outbounds: VLESS, VMess, Trojan и Shadowsocks. Локальные inbounds, DNS и routing не переносятся.</p>
   <Button variant="primary" disabled={!json.trim()||add.isPending} loading={add.isPending} onClick={()=>add.mutate()}><Plus size={17}/> Импортировать серверы</Button>
  </section>
  <section className="mt-5 space-y-3">
   <div className="flex items-center justify-between"><h2 className="font-semibold">Импортированные серверы</h2><Button size="sm" onClick={refresh}><RefreshCw size={15}/> Обновить список</Button></div>
   {list.isLoading?<Skeleton/>:list.isError?<EmptyState title="Не удалось загрузить серверы" text="Обновите страницу или проверьте соединение."/>:!list.data?.length?<EmptyState title="Пока нет внешних нод" text="Вставьте полный Xray JSON выше, чтобы добавить серверы в подписку."/>:<div className="space-y-2">{list.data.map(n=><article key={n.id} className="card p-3 flex flex-wrap items-center gap-3">
    <div className="min-w-0 flex-1"><div className="font-medium truncate">{n.name}</div><div className="text-sm muted">{n.protocol.toUpperCase()} · {n.server}:{n.port}</div></div>
    <Pill tone={n.enabled?"ok":"off"}>{n.enabled?"В подписке":"Выключен"}</Pill>
    <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={n.auto} onChange={e=>patch.mutate({id:n.id,body:{auto:e.target.checked}})}/> Auto</label>
    <Button size="sm" onClick={()=>patch.mutate({id:n.id,body:{enabled:!n.enabled}})}>{n.enabled?"Выключить":"Включить"}</Button>
    <Button size="sm" variant="danger" onClick={()=>{if(confirm(`Удалить ${n.name}?`))remove.mutate(n.id)}}><Trash2 size={15}/></Button>
   </article>)}</div>}
  </section>
 </>;
}
