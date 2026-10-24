#!/usr/bin/env node
import {execFileSync} from 'node:child_process';
import {existsSync,readFileSync,statSync} from 'node:fs';
import {resolve,dirname,sep} from 'node:path';
const files=execFileSync('git',['ls-files','-z'],{encoding:'utf8'}).split('\0').filter(Boolean);
const errors=[];let links=0;
const forbidden=/(^|\/)(node_modules|\.next|\.playwright-mcp|\.playwright-cli|\.impeccable)(\/|$)|(^|\/)(jwt\.key|config\.json|attack-logs\.json|data-sql-audit\.tmp|prompt - for new version\.md)$|\.(exe|db|db-wal|db-shm|bak|tmp|log)$/i;
for(const file of files){
 if(forbidden.test(file)||/^web\/out\//.test(file)||/^server\/webui\/dist\//.test(file))errors.push('Local artifact is tracked: '+file);
 if(!existsSync(file))continue;
 if(statSync(file).size>10*1024*1024)errors.push('Oversized source artifact: '+file);
 if(!/\.(md|go|ts|tsx|mjs|js|yml|yaml|sh|bat|html|css|svg)$/.test(file))continue;
 const text=readFileSync(file,'utf8');
 if(/[\uac00-\ud7af]/u.test(text))errors.push('Unexpected alternate-language content: '+file);
 if(!file.endsWith('.md'))continue;
 const prose=text.replace(/(^|\n)(```|~~~)[^\n]*\n[\s\S]*?\n\2/g,'\n');
 for(const match of prose.matchAll(/!?\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)){
  let ref=match[1].replace(/^<|>$/g,'').split('#')[0].split('?')[0];
  if(!ref||/^[a-z][a-z0-9+.-]*:/i.test(ref)||ref.startsWith('//'))continue;
  try{ref=decodeURIComponent(ref)}catch{errors.push('Malformed document link: '+file);continue;}
  links++;
  const target=resolve(dirname(file),ref),root=resolve('.');
  if(target!==root&&!target.startsWith(root+sep))errors.push('Link leaves repository: '+file+' -> '+ref);
  else if(!existsSync(target))errors.push('Broken document link: '+file+' -> '+ref);
 }
}
for(const file of ['README.md','LICENSE','SECURITY.md','CONTRIBUTING.md','CODE_OF_CONDUCT.md','.github/workflows/ci.yml','.github/workflows/release.yml'])if(!files.includes(file))errors.push('Missing project file: '+file);
console.log(JSON.stringify({trackedFiles:files.length,localLinks:links,errors},null,2));
if(errors.length)process.exitCode=1;
