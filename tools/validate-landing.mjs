#!/usr/bin/env node
import {readFileSync,existsSync} from 'node:fs';
import {resolve,dirname,sep} from 'node:path';
import {fileURLToPath} from 'node:url';
const root=resolve(dirname(fileURLToPath(import.meta.url)),'../landing');
const html=readFileSync(resolve(root,'index.html'),'utf8');
const errors=[];
if(!/<html\s+lang="en"/.test(html))errors.push('Landing language must be English.');
if(/[\uac00-\ud7af]/u.test(html))errors.push('Unexpected alternate-language text.');
const ids=[...html.matchAll(/\bid="([^"]+)"/g)].map(m=>m[1]);
if(new Set(ids).size!==ids.length)errors.push('Duplicate element IDs.');
const refs=new Set();for(const m of html.matchAll(/(?:src|href)="([^"]+)"|srcset="([^"]+)"/g)){if(m[1])refs.add(m[1]);if(m[2])for(const item of m[2].split(','))refs.add(item.trim().split(/\s+/)[0]);}
for(const ref of refs){if(/^(?:https?:|mailto:|data:|\/\/)/.test(ref))continue;if(ref.startsWith('#')){if(ref!=='#'&&!ids.includes(ref.slice(1)))errors.push('Missing anchor: '+ref);continue;}const clean=ref.split(/[?#]/)[0];if(!clean)continue;if(clean.startsWith('/')){errors.push('Use relative asset paths: '+ref);continue;}const file=resolve(root,clean);if(file!==root&&!file.startsWith(root+sep))errors.push('Reference outside landing: '+ref);else if(!existsSync(file))errors.push('Missing local reference: '+ref);}
console.log(JSON.stringify({references:refs.size,anchors:ids.length,errors},null,2));if(errors.length)process.exitCode=1;
