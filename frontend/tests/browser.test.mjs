import test from 'node:test';
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, rm, access } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { once } from 'node:events';

const chromePath = process.env.CHROME_BIN || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
let available = true;
try {
  await access(chromePath);
} catch {
  available = false;
}
const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function waitFor(check, timeout = 15000) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    const result = await check();
    if (result) return result;
    await delay(50);
  }
  throw new Error('Timed out waiting for browser condition');
}

// Uses the installed browser and Node's WebSocket client; no automation dependency.
test(
  'desktop workspace regression suite in headless Chrome',
  { skip: !available, timeout: 90000 },
  async (t) => {
    const profile = await mkdtemp(join(tmpdir(), 'marktyp-browser-'));
    const server = spawn(
      process.execPath,
      ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '4179', '--strictPort'],
      { stdio: 'ignore' },
    );
    const chrome = spawn(
      chromePath,
      [
        '--headless=new',
        '--disable-gpu',
        '--no-first-run',
        '--no-default-browser-check',
        '--remote-debugging-port=0',
        `--user-data-dir=${profile}`,
        'about:blank',
      ],
      { stdio: ['ignore', 'ignore', 'pipe'] },
    );
    let stderr = '';
    chrome.stderr.on('data', (data) => {
      stderr += data;
    });
    t.after(async () => {
      chrome.kill();
      server.kill();
      await Promise.allSettled([once(chrome, 'exit'), once(server, 'exit')]);
      await rm(profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
    });
    await waitFor(async () => {
      try {
        return (await fetch('http://127.0.0.1:4179')).ok;
      } catch {
        return false;
      }
    });
    const endpoint = await waitFor(() => stderr.match(/DevTools listening on (ws:\/\/\S+)/)?.[1]);
    const socket = new WebSocket(endpoint);
    await once(socket, 'open');
    t.after(() => socket.close());
    let id = 0;
    const pending = new Map();
    socket.addEventListener('message', (event) => {
      const message = JSON.parse(event.data);
      if (message.id) {
        const task = pending.get(message.id);
        pending.delete(message.id);
        if (message.error) task.reject(message.error);
        else task.resolve(message.result);
      }
    });
    function send(method, params = {}, sessionId) {
      return new Promise((resolve, reject) => {
        const key = ++id;
        pending.set(key, { resolve, reject });
        socket.send(JSON.stringify({ id: key, method, params, ...(sessionId ? { sessionId } : {}) }));
      });
    }
    const { targetId } = await send('Target.createTarget', { url: 'about:blank' });
    const { sessionId } = await send('Target.attachToTarget', { targetId, flatten: true });
    const call = (method, params) => send(method, params, sessionId);
    async function evaluate(expression) {
      const result = await call('Runtime.evaluate', {
        expression,
        awaitPromise: true,
        returnByValue: true,
        userGesture: true,
      });
      if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails));
      return result.result.value;
    }
    await call('Page.enable');
    await call('Page.addScriptToEvaluateOnNewDocument', {
      source: `
    window.events = {};
    window.exports = [];
    window.savedDrafts = [];
    window.deletedPaths = [];
    window.confirm = () => true;
    window.runtime = {EventsOnMultiple(name, fn) { window.events[name] = fn; return () => delete window.events[name]; }, BrowserOpenURL(url) { window.openedURL = url; }};
    const doc = (path, markdown) => ({id:path,title:markdown.split('\\n')[0].replace(/^# /,''),path,markdown,hasDraft:false});
    window.workspace = {appInfo:{name:'marktyp',version:'0.1.0',tagline:'Test',modes:['Document','Source','Dual']}, config:{version:1,theme:'marktyp',preferredMode:'Document',autosave:false,lastOpenedPath:'/a.md'}, notes:[{id:'a',path:'/a.md',title:'Alpha',updatedLabel:'Today'},{id:'b',path:'/b.md',title:'Beta',updatedLabel:'Today'}],activeDoc:doc('/a.md','# Alpha\\n\\nInitial text')};
    window.go = {main:{App:{
      GetWorkspace:async()=>structuredClone(window.workspace),
      NewDocument:async()=>doc('','# Untitled\\n\\n'),
      OpenDocument:async()=>{throw Error('open cancelled');},
      OpenDocumentAtPath:async(path)=>{window.workspace.activeDoc=doc(path,path==='/a.md'?'# Alpha':'# Beta'); return structuredClone(window.workspace);},
      SaveDocument:async(request)=>{await new Promise(r=>setTimeout(r,window.saveDelay||0)); window.workspace.activeDoc=doc(request.path,request.markdown);return structuredClone(window.workspace);},
      SaveDocumentAs:async(request)=>{window.workspace.activeDoc=doc('/saved.md',request.markdown);return structuredClone(window.workspace);},
      SaveDraft:async(path,markdown)=>{window.savedDrafts.push({path,markdown});},
      UpdatePreferences:async(request)=>{Object.assign(window.workspace.config,request);return structuredClone(window.workspace.config);},
      DeleteNote:async(path)=>{window.deletedPaths.push(path);window.workspace.notes=window.workspace.notes.filter(n=>n.path!==path);return structuredClone(window.workspace);},
      GetDocument:async(path)=>doc(path,'# Other'),
      SelectImage:async()=>{const editor=document.querySelector('[aria-label="Document editor"]');if(editor){const range=document.createRange();range.setStart(editor,0);range.collapse(true);getSelection().removeAllRanges();getSelection().addRange(range);}return '/pictures/photo.png';},
      GetImagePreview:async()=> 'data:image/png;base64,iVBORw0KGgo=',
      ExportHTML:async(request)=>window.exports.push(request),
      ExportPDF:async(request)=>window.exports.push(request),
      RevealNoteInFS:async()=>{}, ForceQuit:async()=>{window.quit=true;}
    }}};
  `,
    });
    await call('Emulation.setDeviceMetricsOverride', {
      width: 1100,
      height: 720,
      deviceScaleFactor: 1,
      mobile: false,
    });
    await call('Page.navigate', { url: 'http://127.0.0.1:4179' });
    await waitFor(() =>
      evaluate(
        `document.querySelector('.toolbar-titlebar__title')?.textContent.trim() === 'Alpha' && !document.querySelector('.app-shell')?.inert`,
      ),
    );
    const action = async (name) => {
      await evaluate(`window.events['marktyp:menu-action'](${JSON.stringify(name)})`);
      await delay(100);
    };
    const input = async (text) => {
      await evaluate(
        `(()=>{const el=document.querySelector('.source-editor');el.focus();el.value=${JSON.stringify(text)};el.setSelectionRange(el.value.length,el.value.length);el.dispatchEvent(new Event('input',{bubbles:true}));})()`,
      );
      await delay(30);
    };

    await t.test('Dual fits the minimum desktop window', async () => {
      await action('view:dual');
      assert.ok(
        await evaluate(
          `(()=>{const el=document.querySelector('.workspace__body');return el.scrollWidth<=el.clientWidth+1;})()`,
        ),
      );
    });
    await t.test('splitters are keyboard adjustable', async () => {
      const before = await evaluate(`Number(document.querySelector('.dual-resizer').dataset.value)`);
      await evaluate(
        `document.querySelector('.dual-resizer').dispatchEvent(new KeyboardEvent('keydown',{bubbles:true,key:'ArrowLeft'}))`,
      );
      assert.equal(
        await evaluate(`Number(document.querySelector('.dual-resizer').dataset.value)`),
        before - 2,
      );
    });
    await t.test('Source menu save includes unapplied source edits', async () => {
      await action('view:source');
      await input('# Source saved');
      await action('file:save');
      assert.equal(await evaluate('window.workspace.activeDoc.markdown'), '# Source saved');
    });
    await t.test('source undo restores the previous editor snapshot', async () => {
      await input('# Undo one');
      await input('# Undo two');
      await evaluate(
        `document.querySelector('.source-editor').dispatchEvent(new KeyboardEvent('keydown',{bubbles:true,key:'z',ctrlKey:true}))`,
      );
      await delay(30);
      assert.equal(await evaluate(`document.querySelector('.source-editor').value`), '# Undo one');
      assert.ok(
        await evaluate(
          `document.activeElement===document.querySelector('.source-editor') && document.activeElement.selectionStart>0`,
        ),
      );
    });
    await t.test('dual undo keeps source focus and caret', async () => {
      await action('view:dual');
      await input('# Dual one');
      await input('# Dual two');
      await evaluate(
        `document.querySelector('.source-editor').dispatchEvent(new KeyboardEvent('keydown',{bubbles:true,key:'z',ctrlKey:true}))`,
      );
      await delay(30);
      assert.equal(await evaluate(`document.querySelector('.source-editor').value`), '# Dual one');
      assert.ok(
        await evaluate(
          `document.activeElement===document.querySelector('.source-editor') && document.activeElement.selectionStart>0`,
        ),
      );
    });
    await t.test('save response does not overwrite typing performed during I/O', async () => {
      await input('# First revision');
      await evaluate('window.saveDelay=250');
      await action('file:save');
      await input('# Newer revision');
      await delay(300);
      assert.equal(await evaluate(`document.querySelector('.source-editor').value`), '# Newer revision');
      assert.equal(await evaluate('window.workspace.activeDoc.markdown'), '# First revision');
      await evaluate('window.saveDelay=0');
    });
    await t.test('cancelled open preserves current edits', async () => {
      await action('file:open');
      assert.equal(await evaluate(`document.querySelector('.source-editor').value`), '# Newer revision');
    });
    await t.test('switching note flushes the latest source draft', async () => {
      await evaluate(
        `Array.from(document.querySelectorAll('.note-card')).find(el=>el.textContent.includes('Beta')).click()`,
      );
      await waitFor(() =>
        evaluate(`document.querySelector('.toolbar-titlebar__title').textContent.trim()==='Beta'`),
      );
      assert.ok(
        await evaluate(`window.savedDrafts.some(d=>d.path==='/a.md' && d.markdown==='# Newer revision')`),
      );
    });
    await t.test('export uses current source rather than a stale preview', async () => {
      await input('# Latest export');
      await action('export:html');
      await waitFor(() => evaluate('window.exports.length > 0'));
      assert.match(await evaluate('window.exports.at(-1).html'), /Latest export/);
    });
    await t.test('Markdown sanitizer and code/math round trips', async () => {
      const result = await evaluate(`(async()=>{
      const {markdownToHtml,htmlToMarkdown}=await import('/src/lib/utils/markdown.ts');
      const code=markdownToHtml('\x60\x60\x60js\\nconst price = "$5$";\\n\x60\x60\x60');
      const inline=markdownToHtml('Use \x60$x$\x60 literally');
      const unsafe=markdownToHtml('<img src="x" onerror="window.injected=true"><script>window.injected=true</script><a href="javascript:alert(1)">bad</a>');
      const task=htmlToMarkdown(markdownToHtml('- [x] Done\\n- [ ] Pending'));
      const localImageMarkdown=htmlToMarkdown('<p><img alt="photo" src="data:image/png;base64,preview" data-marktyp-source="/pictures/My photo.png"></p>');
      const localImage=markdownToHtml(localImageMarkdown);
      return {code,inline,unsafe,task,localImage,localImageMarkdown};
    })()`);
      assert.ok(!result.code.includes('marktyp-math'));
      assert.ok(!result.inline.includes('marktyp-math'));
      assert.ok(!/onerror|<script|javascript:/i.test(result.unsafe));
      assert.match(result.task, /\[x\] Done/);
      assert.match(result.task, /\[ \] Pending/);
      assert.equal(result.localImageMarkdown, '![photo](</pictures/My photo.png>)');
      assert.match(result.localImage, /data-marktyp-source="\/pictures\/My photo\.png"/);
    });
    await t.test('task checkboxes remain usable in the visual editor', async () => {
      await input('# Tasks\n\n- [ ] Pending');
      await action('view:document');
      await waitFor(() => evaluate(`!!document.querySelector('input[type=checkbox]')`));
      await evaluate(`document.querySelector('input[type=checkbox]').click()`);
      await action('view:source');
      assert.match(await evaluate(`document.querySelector('.source-editor').value`), /\[x\]\s+Pending/);
    });
    await t.test('pasted rich HTML is sanitized before insertion', async () => {
      await action('view:document');
      const result = await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');editor.focus();const range=document.createRange();range.selectNodeContents(editor);range.collapse(false);getSelection().removeAllRanges();getSelection().addRange(range);const data=new DataTransfer();data.setData('text/html','<img src="x" onerror="window.injected=true"><b>safe</b>');editor.dispatchEvent(new ClipboardEvent('paste',{bubbles:true,clipboardData:data}));return {html:editor.innerHTML,injected:window.injected};})()`,
      );
      assert.ok(!result.injected);
      assert.ok(!/onerror/i.test(result.html));
      assert.match(result.html, /safe/);
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'z',
        code: 'KeyZ',
        windowsVirtualKeyCode: 90,
        modifiers: 2,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'z',
        code: 'KeyZ',
        windowsVirtualKeyCode: 90,
        modifiers: 2,
      });
      await delay(30);
      assert.ok(
        !(await evaluate(
          `document.querySelector('[aria-label="Document editor"]').textContent.includes('safe')`,
        )),
      );
      assert.ok(
        await evaluate(
          `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const selection=getSelection();const before=document.createRange();before.selectNodeContents(editor);before.setEnd(selection.anchorNode,selection.anchorOffset);return document.activeElement===editor && editor.contains(selection.anchorNode) && before.toString().length>0;})()`,
        ),
      );
    });
    await t.test('inline backticks become inline code in Document mode', async () => {
      await action('view:source');
      await input('# Code\n\nParagraph');
      await action('view:document');
      await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const p=editor.querySelector('p');editor.focus();const r=document.createRange();r.selectNodeContents(p);r.collapse(false);getSelection().removeAllRanges();getSelection().addRange(r);})()`,
      );
      await call('Input.insertText', { text: ' `cane`' });
      await delay(50);
      assert.equal(
        await evaluate(`document.querySelector('[aria-label="Document editor"] code')?.textContent`),
        'cane',
      );
      assert.ok(
        await evaluate(`document.activeElement===document.querySelector('[aria-label="Document editor"]')`),
      );
      await action('view:source');
      assert.match(await evaluate(`document.querySelector('.source-editor').value`), /`cane`/);
    });
    await t.test('inline code renders immediately when typed one character at a time', async () => {
      await action('view:source');
      await input('# Immediate code\n\nParagraph');
      await action('view:document');
      await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const p=editor.querySelector('p');editor.focus();const r=document.createRange();r.selectNodeContents(p);r.collapse(false);getSelection().removeAllRanges();getSelection().addRange(r);})()`,
      );
      for (const character of ' `gatto`') await call('Input.insertText', { text: character });
      assert.equal(
        await evaluate(`document.querySelector('[aria-label="Document editor"] code')?.textContent`),
        'gatto',
      );
      assert.equal(
        await evaluate(
          `document.querySelector('[aria-label="Document editor"]').querySelectorAll('code').length`,
        ),
        1,
      );
    });
    await t.test('Markdown renders from the root of an empty Document editor', async () => {
      await action('view:source');
      await input('');
      await action('view:document');
      await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');editor.focus();editor.textContent='# cina';const r=document.createRange();r.setStart(editor,editor.childNodes.length);r.collapse(true);getSelection().removeAllRanges();getSelection().addRange(r);editor.dispatchEvent(new InputEvent('input',{bubbles:true,inputType:'insertText',data:'a'}));})()`,
      );
      await delay(30);
      assert.equal(
        await evaluate(`document.querySelector('[aria-label="Document editor"] h1')?.textContent`),
        'cina',
      );
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.insertText', { text: '`cane`' });
      await delay(30);
      assert.equal(
        await evaluate(`document.querySelector('[aria-label="Document editor"] code')?.textContent`),
        'cane',
      );
      await call('Input.insertText', { text: ' dopo' });
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.insertText', { text: 'riga successiva' });
      assert.equal(
        await evaluate(`document.querySelector('[aria-label="Document editor"] code')?.textContent`),
        'cane',
      );
      assert.match(
        await evaluate(`document.querySelector('[aria-label="Document editor"]').textContent`),
        /dopo[\s\S]*riga successiva/,
      );
    });
    await t.test('Ctrl+A selects the complete Document editor', async () => {
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'a',
        code: 'KeyA',
        modifiers: 2,
        windowsVirtualKeyCode: 65,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'a',
        code: 'KeyA',
        modifiers: 2,
        windowsVirtualKeyCode: 65,
      });
      assert.ok(
        await evaluate(
          `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const range=getSelection().getRangeAt(0);return range.startContainer===editor&&range.startOffset===0&&range.endContainer===editor&&range.endOffset===editor.childNodes.length;})()`,
        ),
      );
    });
    await t.test('typing a triple-backtick fence creates a code block', async () => {
      await action('view:source');
      await input('# Fence\n\nParagraph');
      await action('view:document');
      await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const p=editor.querySelector('p');editor.focus();p.innerHTML='<br>';const r=document.createRange();r.setStart(p,0);r.collapse(true);getSelection().removeAllRanges();getSelection().addRange(r);})()`,
      );
      await call('Input.insertText', { text: '```' });
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await delay(30);
      const fenceState = await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');return {html:editor.innerHTML,anchor:getSelection().anchorNode?.nodeName,text:getSelection().anchorNode?.textContent}})()`,
      );
      assert.ok(fenceState.html.includes('<pre>'), JSON.stringify(fenceState));
      assert.ok(
        await evaluate(
          `document.querySelector('[aria-label="Document editor"] pre code').contains(getSelection().anchorNode)`,
        ),
      );
    });
    await t.test('typing a closing fence exits without creating another code block', async () => {
      await call('Input.insertText', { text: 'inside' });
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      for (let index = 0; index < 3; index += 1) {
        await call('Input.dispatchKeyEvent', {
          type: 'keyDown',
          key: '`',
          code: 'Backquote',
          windowsVirtualKeyCode: 192,
        });
        await call('Input.insertText', { text: '`' });
        await call('Input.dispatchKeyEvent', {
          type: 'keyUp',
          key: '`',
          code: 'Backquote',
          windowsVirtualKeyCode: 192,
        });
      }
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      const closingFenceState = await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const anchor=getSelection().anchorNode;const element=anchor instanceof Element?anchor:anchor?.parentElement;return {outside:!element?.closest('pre'),html:editor.innerHTML,text:editor.textContent,anchor:element?.outerHTML};})()`,
      );
      assert.ok(closingFenceState.outside, JSON.stringify(closingFenceState));
      await call('Input.insertText', { text: 'outside-fence' });
      await action('view:source');
      const markdown = await evaluate(`document.querySelector('.source-editor').value`);
      assert.equal(markdown.match(/^```/gm)?.length, 2, markdown);
      assert.match(markdown, /```txt[\s\S]*inside[\s\S]*```[\s\S]*outside-fence/);
      await action('view:document');
    });
    await t.test('toolbar code blocks place the caret inside code', async () => {
      await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');editor.focus();const r=document.createRange();r.selectNodeContents(editor);r.collapse(false);getSelection().removeAllRanges();getSelection().addRange(r);Array.from(document.querySelectorAll('summary')).find(el=>el.textContent.includes('Code')).click();})()`,
      );
      await evaluate(
        `Array.from(document.querySelectorAll('.toolbar-menu__item')).find(el=>el.textContent.trim()==='Code Block').click()`,
      );
      assert.ok(
        await evaluate(
          `Array.from(document.querySelectorAll('[aria-label="Document editor"] pre code')).some(code=>code.contains(getSelection().anchorNode))`,
        ),
      );
    });
    await t.test('double Enter exits a code block', async () => {
      await call('Input.insertText', { text: 'inside' });
      await evaluate(
        `(()=>{const anchor=getSelection().anchorNode;const element=anchor instanceof Element?anchor:anchor.parentElement;const code=element.closest('pre code');const pre=code.parentElement;const range=document.createRange();range.setStart(pre,pre.childNodes.length);range.collapse(true);getSelection().removeAllRanges();getSelection().addRange(range);})()`,
      );
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      assert.ok(
        await evaluate(
          `Array.from(document.querySelectorAll('[aria-label="Document editor"] pre code')).some(code=>code.contains(getSelection().anchorNode))`,
        ),
      );
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'Enter',
        code: 'Enter',
        windowsVirtualKeyCode: 13,
      });
      await delay(30);
      const exitedCode = await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const anchor=getSelection().anchorNode;const element=anchor instanceof Element?anchor:anchor?.parentElement;return {ok:!!anchor&&editor.contains(anchor)&&!element?.closest('pre'),html:editor.innerHTML,anchor:element?.outerHTML};})()`,
      );
      assert.ok(exitedCode.ok, JSON.stringify(exitedCode));
      await call('Input.insertText', { text: 'outside' });
      await action('view:source');
      const markdown = await evaluate(`document.querySelector('.source-editor').value`);
      assert.match(markdown, /```txt[\s\S]*inside[\s\S]*```[\s\S]*outside/);
    });
    await t.test('Insert supports tables at the current cursor position', async () => {
      await action('view:document');
      await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const p=Array.from(editor.querySelectorAll('p')).find(node=>node.textContent.includes('outside'));editor.focus();const r=document.createRange();r.selectNodeContents(p);r.collapse(false);getSelection().removeAllRanges();getSelection().addRange(r);})()`,
      );
      await delay(30);
      await evaluate(
        `Array.from(document.querySelectorAll('summary')).find(el=>el.textContent.includes('Insert')).click()`,
      );
      await evaluate(
        `Array.from(document.querySelectorAll('.toolbar-menu__item')).find(el=>el.textContent.trim()==='Table').click()`,
      );
      await waitFor(() => evaluate(`!!document.querySelector('.table-dialog[open]')`));
      await evaluate(
        `Array.from(document.querySelectorAll('.table-dialog button')).find(el=>el.textContent.trim()==='Insert').click()`,
      );
      assert.equal(
        await evaluate(`document.querySelectorAll('[aria-label="Document editor"] table th').length`),
        3,
      );
      assert.ok(
        await evaluate(
          `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const p=Array.from(editor.querySelectorAll('p')).find(node=>node.textContent.includes('outside'));return p?.nextElementSibling?.tagName==='TABLE';})()`,
        ),
      );
      assert.ok(
        await evaluate(
          `document.querySelector('[aria-label="Document editor"] th').contains(getSelection().anchorNode)`,
        ),
      );
      await action('view:source');
      assert.match(
        await evaluate(`document.querySelector('.source-editor').value`),
        /\| Column 1 \| Column 2 \| Column 3 \|/,
      );
      await action('view:document');
    });
    await t.test('Insert Image preserves the cursor across the native dialog', async () => {
      await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const p=Array.from(editor.querySelectorAll('p')).find(node=>node.textContent.includes('outside'));editor.focus();const r=document.createRange();r.selectNodeContents(p);r.collapse(false);getSelection().removeAllRanges();getSelection().addRange(r);})()`,
      );
      await delay(30);
      await evaluate(
        `Array.from(document.querySelectorAll('summary')).find(el=>el.textContent.includes('Insert')).click()`,
      );
      await evaluate(
        `Array.from(document.querySelectorAll('.toolbar-menu__item')).find(el=>el.textContent.trim()==='Image').click()`,
      );
      await waitFor(() =>
        evaluate(
          `document.querySelector('[aria-label="Document editor"] img')?.src.startsWith('data:image/png')`,
        ),
      );
      assert.equal(
        await evaluate(`document.querySelector('[aria-label="Document editor"] img')?.dataset.marktypSource`),
        '/pictures/photo.png',
      );
      assert.ok(
        await evaluate(
          `!!Array.from(document.querySelectorAll('[aria-label="Document editor"] p')).find(node=>node.textContent.includes('outside'))?.querySelector('img')`,
        ),
      );
      await evaluate(
        `(()=>{const editor=document.querySelector('[aria-label="Document editor"]');const p=Array.from(editor.querySelectorAll('p')).find(node=>node.textContent.includes('outside'));editor.focus();const r=document.createRange();r.selectNodeContents(p);r.collapse(false);getSelection().removeAllRanges();getSelection().addRange(r);})()`,
      );
      await call('Input.insertText', { text: ' after-image' });
      await evaluate(`document.querySelector('.note-search').focus()`);
      await delay(100);
      assert.ok(
        await evaluate(
          `document.querySelector('[aria-label="Document editor"] img')?.src.startsWith('data:image/png')`,
        ),
      );
      assert.ok(
        !(await evaluate(
          `document.querySelector('[aria-label="Document editor"]').textContent.includes('![image]')`,
        )),
      );
      await action('view:source');
      assert.match(
        await evaluate(`document.querySelector('.source-editor').value`),
        /!\[image\]\(\/pictures\/photo\.png\)/,
      );
    });
    await t.test('external links use the native browser bridge', async () => {
      await action('view:source');
      await input('# Link\n\n[site](https://example.com)');
      await action('view:document');
      await waitFor(() => evaluate(`!!document.querySelector('.document-preview a')`));
      await evaluate(
        `document.querySelector('.document-preview a').dispatchEvent(new MouseEvent('click',{bubbles:true,ctrlKey:true}))`,
      );
      assert.match(await evaluate('window.openedURL'), /^https:\/\/example\.com/);
    });
    await t.test('failed autosave is not retried forever without a new edit', async () => {
      await action('view:source');
      await evaluate(
        `window.originalSave=window.go.main.App.SaveDocument;window.saveCalls=0;window.go.main.App.SaveDocument=async()=>{window.saveCalls++;throw Error('disk full')}`,
      );
      await action('view:toggle-autosave');
      await input('# Autosave failure');
      await delay(3000);
      assert.equal(await evaluate('window.saveCalls'), 1);
      await action('view:toggle-autosave');
      await evaluate(`window.go.main.App.SaveDocument=window.originalSave`);
    });
    await t.test('Mermaid can be edited and Escape closes the modal', async () => {
      await input('# Diagram\n\n```mermaid\ngraph TD; A-->B\n```');
      await action('view:document');
      await waitFor(() => evaluate(`!!document.querySelector('marktyp-mermaid')`));
      await evaluate(`document.querySelector('marktyp-mermaid').click()`);
      await waitFor(() => evaluate(`!!document.querySelector('dialog[open]')`));
      assert.equal(await evaluate(`document.activeElement.className`), 'rich-editor-modal__input');
      await call('Input.dispatchKeyEvent', {
        type: 'keyDown',
        key: 'Escape',
        code: 'Escape',
        windowsVirtualKeyCode: 27,
      });
      await call('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: 'Escape',
        code: 'Escape',
        windowsVirtualKeyCode: 27,
      });
      await waitFor(() => evaluate(`!document.querySelector('dialog[open]')`));
    });
    await t.test('context menu remains within viewport', async () => {
      await evaluate(
        `document.querySelector('.note-card').dispatchEvent(new MouseEvent('contextmenu',{bubbles:true,clientX:1098,clientY:718}))`,
      );
      await delay(50);
      assert.ok(
        await evaluate(
          `(()=>{const r=document.querySelector('.note-context-menu').getBoundingClientRect();return r.right<=innerWidth && r.bottom<=innerHeight;})()`,
        ),
      );
    });
    await t.test('context menu deletes the selected sidebar note', async () => {
      const before = await evaluate(`document.querySelectorAll('.note-card').length`);
      const deletedPath = await evaluate(
        `(()=>{const note=Array.from(document.querySelectorAll('.note-card')).find(node=>node.getAttribute('aria-current')!=='page');const path=note.title;note.dispatchEvent(new MouseEvent('contextmenu',{bubbles:true,clientX:100,clientY:100}));return path;})()`,
      );
      await waitFor(() => evaluate(`!!document.querySelector('.note-context-menu')`));
      await evaluate(`document.querySelector('.note-context-menu .is-danger').click()`);
      await waitFor(() => evaluate(`!!document.querySelector('.delete-note-dialog[open]')`));
      await evaluate(`document.querySelector('.delete-note-dialog .ghost-button--danger').click()`);
      await waitFor(() => evaluate(`window.deletedPaths.includes(${JSON.stringify(deletedPath)})`));
      assert.ok(await evaluate(`window.deletedPaths.includes(${JSON.stringify(deletedPath)})`));
      assert.equal(await evaluate(`document.querySelectorAll('.note-card').length`), before - 1);
      assert.ok(
        !(await evaluate(
          `Array.from(document.querySelectorAll('.note-card')).some(node=>node.title===${JSON.stringify(deletedPath)})`,
        )),
      );
    });
    await t.test('close flushes dirty edits before quitting', async () => {
      await evaluate(`window.events['marktyp:request-close']()`);
      await waitFor(() => evaluate('window.quit === true'));
      assert.ok(await evaluate(`window.savedDrafts.some(d=>d.markdown.includes('graph TD'))`));
    });
  },
);
