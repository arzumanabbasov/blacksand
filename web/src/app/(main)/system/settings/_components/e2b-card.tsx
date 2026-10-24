"use client";
import * as React from "react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function E2BCard() {
 const [key, setKey] = React.useState("");
 const [template, setTemplate] = React.useState("");
 const [configured, setConfigured] = React.useState(false);
 const [saving, setSaving] = React.useState(false);
 React.useEffect(() => { api.e2bSettings().then(s => { setConfigured(s.key_set); setTemplate(s.template); }).catch(e => toast.error(e.message)); }, []);
 async function save() {
  setSaving(true);
  try {
   const s = await api.saveE2BSettings({template:template.trim(), ...(key.trim() ? {api_key:key.trim()} : {})});
   setConfigured(s.key_set); setKey("");
   toast.success("E2B enabled. Start a new agent run to use the updated settings.");
  } catch(e) { toast.error((e as Error).message); }
  finally { setSaving(false); }
 }
 return <Card className="mb-4 break-inside-avoid md:mb-6">
  <CardHeader><CardTitle>E2B execution</CardTitle><CardDescription>Run agent commands and generated code in a remote Linux sandbox with internet access.</CardDescription></CardHeader>
  <CardContent className="space-y-4">
   <div className="space-y-2"><Label htmlFor="e2b-key">E2B API key {configured ? "(configured)" : "(required)"}</Label><Input id="e2b-key" type="password" autoComplete="new-password" value={key} onChange={e => setKey(e.target.value)} placeholder={configured ? "Leave blank to keep saved key" : "Enter your E2B API key"}/></div>
   <div className="space-y-2"><Label htmlFor="e2b-template">Sandbox template</Label><Input id="e2b-template" value={template} onChange={e => setTemplate(e.target.value)} placeholder="base"/></div>
   <p className="text-sm text-muted-foreground">Internet access is enabled. Commands can install packages and access sandbox files. E2B account limits apply. The key is stored in the backend database.</p>
   <Button disabled={saving || (!configured && !key.trim())} onClick={save}>{saving ? "Saving…" : "Save and enable E2B"}</Button>
  </CardContent>
 </Card>;
}
