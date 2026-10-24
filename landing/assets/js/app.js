'use strict';
const button=document.getElementById('copy-command');
const status=document.getElementById('copy-status');
if(button)button.addEventListener('click',async()=>{try{await navigator.clipboard.writeText(document.getElementById('install-command').textContent);button.textContent='Copied';status.textContent='Setup commands copied to clipboard.';setTimeout(()=>{button.textContent='Copy'},1800)}catch{status.textContent='Clipboard unavailable. Select and copy the commands manually.';button.textContent='Select text';const selection=window.getSelection();const range=document.createRange();range.selectNodeContents(document.getElementById('install-command'));selection.removeAllRanges();selection.addRange(range);}});
