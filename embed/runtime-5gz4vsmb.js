var N=0,S=0,W=0,x=new Set;var X=new Set,I=null;if(typeof globalThis.FinalizationRegistry<"u")I=new globalThis.FinalizationRegistry((F)=>{});var B=!1;function g(F=!0){B=F}function h(){let F=0;for(let G of X)if(G.deref())F++;return F}function m(){for(let F of X){let G=F.deref();if(G&&!G.isDisposed())G.dispose()}X.clear()}function w(F){if(B){if(typeof globalThis.WeakRef<"u"){let G=globalThis.WeakRef;X.add(new G(F))}else X.add({deref:()=>F});if(I)I.register(F,`disposable-${Date.now()}`)}return F}var $=null,M=[];function j(F,G){if(F===G)return!0;if(typeof F!==typeof G)return!1;if(typeof F!=="object"||F===null||G===null)return!1;if(Array.isArray(F)&&Array.isArray(G)){if(F.length!==G.length)return!1;for(let J=0;J<F.length;J++)if(!j(F[J],G[J]))return!1;return!0}if(Array.isArray(F)!==Array.isArray(G))return!1;let L=Object.keys(F),H=Object.keys(G);if(L.length!==H.length)return!1;for(let J of L){if(!Object.prototype.hasOwnProperty.call(G,J))return!1;if(!j(F[J],G[J]))return!1}return!0}function E(F){W++;try{F()}finally{if(W--,W===0){let G=[...x];x.clear(),G.forEach((L)=>L.notify())}}}class C{_value;_id;_subscribers=new Set;_dirty=!1;_disposed=!1;_hasPendingOldValue=!1;_pendingOldValue;constructor(F){this._value=F,this._id=++N,w(this)}get value(){return this.trackDependency(),this._value}set value(F){if(this._equal(this._value,F))return;let G=this._value;this._value=F,this._dirty=!0,this._notifySubscribers(G)}get(){return this.trackDependency(),this._value}set(F){this.value=F}update(F){this.value=F(this._value)}subscribe(F){return this._subscribers.add(F),()=>this._subscribers.delete(F)}_notifySubscribers(F){if(!this._hasPendingOldValue)this._hasPendingOldValue=!0,this._pendingOldValue=F;if(W>0){x.add(this);return}this.notify(F)}notify(F){let G=this._value,L=this._hasPendingOldValue?this._pendingOldValue:F!==void 0?F:G;this._hasPendingOldValue=!1,this._pendingOldValue=void 0,this._subscribers.forEach((H)=>H(G,L))}_equal(F,G){if(Object.is(F,G))return!0;if(typeof F!==typeof G)return!1;if(typeof F!=="object"||F===null||G===null)return!1;return j(F,G)}trackDependency(){if($)$.addDependency(this)}toJSON(){return{id:this._id,value:this._value}}dispose(){this._disposed=!0,this._subscribers.clear()}isDisposed(){return this._disposed}}class R{_value;_compute;_dependencies=new Set;_subscribers=new Set;_depUnsubs=new Map;_dirty=!0;_disposed=!1;constructor(F){this._compute=F,this._value=void 0,this._recompute()}get value(){if(this._dirty)this._recompute();return this.trackDependency(),this._value}get(){return this.value}subscribe(F){return this._subscribers.add(F),()=>this._subscribers.delete(F)}_recompute(){let F=new Set(this._dependencies);this._dependencies.clear();let G=$;$={addDependency:(H)=>{this._dependencies.add(H)}};try{this._value=this._compute(),this._dirty=!1}finally{$=G}F.forEach((H)=>{if(!this._dependencies.has(H)){let J=this._depUnsubs.get(H);if(J)J(),this._depUnsubs.delete(H)}}),this._dependencies.forEach((H)=>{if(!F.has(H)){let J=H.subscribe(()=>{this._dirty=!0,this._notifySubscribers()});this._depUnsubs.set(H,J)}})}_notifySubscribers(){if(W>0){x.add(this);return}this.notify()}notify(){let F=this._dirty?void 0:this._value;if(this._dirty)this._recompute();let G=this._value;this._subscribers.forEach((L)=>L(G,F??G))}trackDependency(){if($)$.addDependency(this)}dispose(){this._disposed=!0,this._depUnsubs.forEach((F)=>F()),this._depUnsubs.clear(),this._dependencies.clear(),this._subscribers.clear()}isDisposed(){return this._disposed}}class O{_fn;_cleanup;_dependencies=new Set;_depUnsubs=new Map;_id;_active=!0;_disposed=!1;constructor(F){this._fn=F,this._id=++S,this._cleanup=void 0,this._run()}_run(){if(!this._active||this._disposed)return;if(this._cleanup)this._cleanup(),this._cleanup=void 0;let F=new Set(this._dependencies);this._dependencies.clear(),M.push(this),$=this;try{this._cleanup=this._fn()}finally{M.pop(),$=M[M.length-1]||null}F.forEach((G)=>{if(!this._dependencies.has(G)){let L=this._depUnsubs.get(G);if(L)L(),this._depUnsubs.delete(G)}}),this._dependencies.forEach((G)=>{if(!F.has(G)){let L=G.subscribe(()=>this.notify());this._depUnsubs.set(G,L)}})}addDependency(F){this._dependencies.add(F)}notify(){this._run()}pause(){this._active=!1}resume(){this._active=!0,this._run()}dispose(){if(this._cleanup)this._cleanup();this._disposed=!0,this._depUnsubs.forEach((F)=>F()),this._depUnsubs.clear(),this._dependencies.clear()}isDisposed(){return this._disposed}}function b(F){return new O(F)}function p(F,G){let L=Array.isArray(F)?F:[F],H=[],J=L.map((Q)=>Q.get());return L.forEach((Q)=>{H.push(Q.subscribe(()=>{let Z=L.map((Y)=>Y.get()),_=J;J=[...Z],G(Array.isArray(F)?Z:Z[0],Array.isArray(F)?_:_[0])}))}),()=>H.forEach((Q)=>Q())}class T{_runes=new Map;set(F,G){let L=this._runes.get(F);if(L)return L.set(G),L;let H=new C(G);return this._runes.set(F,H),H}get(F){return this._runes.get(F)}has(F){return this._runes.has(F)}delete(F){return this._runes.delete(F)}clear(){this._runes.clear()}toJSON(){let F={};return this._runes.forEach((G,L)=>{F[L]=G.get()}),F}fromJSON(F){Object.entries(F).forEach(([G,L])=>{if(this._runes.has(G))this._runes.get(G).set(L);else this.set(G,L)})}dispose(){this._runes.forEach((F)=>{if("dispose"in F&&typeof F.dispose==="function")F.dispose()}),this._runes.clear()}isDisposed(){return this._runes.size===0}}var U=null,z=!1;function k(){if(!K()||z)return;z=!0,U=document.createElement("div"),U.id="gospa-devtools",U.innerHTML=`
		<style>
			#gospa-devtools {
				position: fixed;
				bottom: 0;
				right: 0;
				width: 320px;
				max-height: 400px;
				background: #1a1a2e;
				color: #eee;
				font-family: 'SF Mono', 'Fira Code', monospace;
				font-size: 12px;
				border-top-left-radius: 8px;
				box-shadow: -4px -4px 20px rgba(0,0,0,0.3);
				z-index: 99999;
				overflow: hidden;
				display: flex;
				flex-direction: column;
			}
			#gospa-devtools-header {
				display: flex;
				justify-content: space-between;
				align-items: center;
				padding: 8px 12px;
				background: #16213e;
				border-bottom: 1px solid #0f3460;
				cursor: move;
			}
			#gospa-devtools-header span {
				font-weight: bold;
				color: #e94560;
			}
			#gospa-devtools-header button {
				background: none;
				border: none;
				color: #888;
				cursor: pointer;
				font-size: 16px;
				padding: 0 4px;
			}
			#gospa-devtools-header button:hover {
				color: #fff;
			}
			#gospa-devtools-tabs {
				display: flex;
				background: #16213e;
				border-bottom: 1px solid #0f3460;
			}
			#gospa-devtools-tabs button {
				flex: 1;
				background: none;
				border: none;
				color: #888;
				padding: 8px;
				cursor: pointer;
				font-size: 11px;
				text-transform: uppercase;
				letter-spacing: 0.5px;
			}
			#gospa-devtools-tabs button.active {
				color: #e94560;
				border-bottom: 2px solid #e94560;
			}
			#gospa-devtools-content {
				flex: 1;
				overflow-y: auto;
				padding: 8px;
			}
			.gospa-devtools-section {
				margin-bottom: 12px;
			}
			.gospa-devtools-section-title {
				color: #e94560;
				font-weight: bold;
				margin-bottom: 4px;
				font-size: 11px;
				text-transform: uppercase;
				letter-spacing: 0.5px;
			}
			.gospa-devtools-item {
				padding: 4px 8px;
				margin: 2px 0;
				background: #16213e;
				border-radius: 4px;
				font-size: 11px;
			}
			.gospa-devtools-item:hover {
				background: #0f3460;
			}
			.gospa-devtools-key {
				color: #00d9ff;
			}
			.gospa-devtools-value {
				color: #a8ff60;
			}
			.gospa-devtools-error {
				color: #ff6b6b;
			}
			.gospa-devtools-metric {
				display: flex;
				justify-content: space-between;
				padding: 4px 8px;
				margin: 2px 0;
				background: #16213e;
				border-radius: 4px;
			}
			.gospa-devtools-metric-label {
				color: #888;
			}
			.gospa-devtools-metric-value {
				color: #a8ff60;
				font-weight: bold;
			}
		</style>
		<div id="gospa-devtools-header">
			<span>GoSPA DevTools</span>
			<button id="gospa-devtools-close">×</button>
		</div>
		<div id="gospa-devtools-tabs">
			<button class="active" data-tab="components">Components</button>
			<button data-tab="state">State</button>
			<button data-tab="performance">Performance</button>
		</div>
		<div id="gospa-devtools-content">
			<div id="gospa-devtools-components" class="gospa-devtools-tab-content active"></div>
			<div id="gospa-devtools-state" class="gospa-devtools-tab-content" style="display:none"></div>
			<div id="gospa-devtools-performance" class="gospa-devtools-tab-content" style="display:none"></div>
		</div>
	`,document.body.appendChild(U),U.querySelector("#gospa-devtools-close")?.addEventListener("click",()=>{U?.remove(),U=null,z=!1});let G=U.querySelectorAll("#gospa-devtools-tabs button");G.forEach((Z)=>{Z.addEventListener("click",()=>{G.forEach((q)=>q.classList.remove("active")),Z.classList.add("active");let _=Z.getAttribute("data-tab");U?.querySelectorAll(".gospa-devtools-tab-content")?.forEach((q)=>{q.style.display=q.id===`gospa-devtools-${_}`?"block":"none"})})});let L=U.querySelector("#gospa-devtools-header"),H=!1,J=0,Q=0;L?.addEventListener("mousedown",(Z)=>{let _=Z;H=!0,J=_.clientX-(U?.offsetLeft||0),Q=_.clientY-(U?.offsetTop||0)}),document.addEventListener("mousemove",(Z)=>{if(H&&U){let _=Z;U.style.left=`${_.clientX-J}px`,U.style.top=`${_.clientY-Q}px`,U.style.right="auto",U.style.bottom="auto"}}),document.addEventListener("mouseup",()=>{H=!1}),console.log("%c[GoSPA DevTools] Panel initialized","color: #e94560")}function f(){if(!U||!K())return;let F=U.querySelector("#gospa-devtools-components");if(F){let H=window.__GOSPA__?.components;if(H){let J='<div class="gospa-devtools-section">';J+='<div class="gospa-devtools-section-title">Components</div>';for(let[Q,Z]of H){let _=Z.states?Array.from(Z.states.keys()):[];J+=`<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${Q}</span>
					<span class="gospa-devtools-value">(${_.length} states)</span>
				</div>`}J+="</div>",F.innerHTML=J}}let G=U.querySelector("#gospa-devtools-state");if(G){let H=window.__GOSPA__?.globalState;if(H){let J='<div class="gospa-devtools-section">';J+='<div class="gospa-devtools-section-title">Global State</div>';let Q=H.toJSON?H.toJSON():{};for(let[Z,_]of Object.entries(Q)){let Y=typeof _==="object"?JSON.stringify(_):String(_);J+=`<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${Z}:</span>
					<span class="gospa-devtools-value">${Y}</span>
				</div>`}J+="</div>",G.innerHTML=J}}let L=U.querySelector("#gospa-devtools-performance");if(L){let H='<div class="gospa-devtools-section">';if(H+='<div class="gospa-devtools-section-title">Performance Metrics</div>',"memory"in performance&&performance.memory){let Q=performance.memory,Z=(Q.usedJSHeapSize/1024/1024).toFixed(2),_=(Q.totalJSHeapSize/1024/1024).toFixed(2);H+=`<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Heap Used</span>
				<span class="gospa-devtools-metric-value">${Z}MB / ${_}MB</span>
			</div>`}let J=performance.getEntriesByType("measure");if(J.length>0){let Q=J[J.length-1];H+=`<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Last Measure</span>
				<span class="gospa-devtools-metric-value">${Q.name}: ${Q.duration.toFixed(2)}ms</span>
			</div>`}H+="</div>",L.innerHTML=H}}function c(){if(!K())return;if(U)U.remove(),U=null,z=!1;else k()}function K(){return typeof window<"u"&&window.__GOSPA_DEV__!==!1}function A(...F){if(!K())return{with:()=>{}};let G=!0,L=[],H=()=>F.map((Q)=>typeof Q==="function"?Q():Q),J=(Q)=>{let Z=H();console.log(`%c[${Q}]`,"color: #888",...Z),L.forEach((_)=>_(Q,Z))};return new O(()=>{if(H(),G)G=!1,J("init");else J("update")}),{with:(Q)=>{L.push(Q)}}}A.trace=(F)=>{if(!K())return;console.log(`%c[trace]${F?` ${F}`:""}`,"color: #666; font-style: italic")};
export{k as ta,f as ua,c as va,g as wa,h as xa,m as ya,E as za,C as Aa,R as Ba,O as Ca,b as Da,p as Ea,T as Fa};
