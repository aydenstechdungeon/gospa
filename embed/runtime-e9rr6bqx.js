var N=0,S=0,Y=0,x=new Set;var q=new Set,I=null;if(typeof globalThis.FinalizationRegistry<"u")I=new globalThis.FinalizationRegistry((F)=>{});var B=!1;function g(F=!0){B=F}function h(){let F=0;for(let G of q)if(G.deref())F++;return F}function m(){for(let F of q){let G=F.deref();if(G&&!G.isDisposed())G.dispose()}q.clear()}function w(F){if(B){if(typeof globalThis.WeakRef<"u"){let G=globalThis.WeakRef;q.add(new G(F))}else q.add({deref:()=>F});if(I)I.register(F,`disposable-${Date.now()}`)}return F}var K=null,M=[];function E(){return K}function j(F,G){if(F===G)return!0;if(typeof F!==typeof G)return!1;if(typeof F!=="object"||F===null||G===null)return!1;if(Array.isArray(F)&&Array.isArray(G)){if(F.length!==G.length)return!1;for(let H=0;H<F.length;H++)if(!j(F[H],G[H]))return!1;return!0}if(Array.isArray(F)!==Array.isArray(G))return!1;let L=Object.keys(F),J=Object.keys(G);if(L.length!==J.length)return!1;for(let H of L){if(!Object.prototype.hasOwnProperty.call(G,H))return!1;if(!j(F[H],G[H]))return!1}return!0}function b(F){Y++;try{F()}finally{if(Y--,Y===0){let G=[...x];x.clear(),G.forEach((L)=>L.notify())}}}class C{_value;_id;_subscribers=new Set;_dirty=!1;_disposed=!1;_hasPendingOldValue=!1;_pendingOldValue;constructor(F){this._value=F,this._id=++N,w(this)}get value(){return this.trackDependency(),this._value}set value(F){if(this._equal(this._value,F))return;let G=this._value;this._value=F,this._dirty=!0,this._notifySubscribers(G)}get(){return this.trackDependency(),this._value}set(F){this.value=F}update(F){this.value=F(this._value)}subscribe(F){return this._subscribers.add(F),()=>this._subscribers.delete(F)}_notifySubscribers(F){if(!this._hasPendingOldValue)this._hasPendingOldValue=!0,this._pendingOldValue=F;if(Y>0){x.add(this);return}this.notify(F)}notify(F){let G=this._value,L=this._hasPendingOldValue?this._pendingOldValue:F!==void 0?F:G;this._hasPendingOldValue=!1,this._pendingOldValue=void 0,this._subscribers.forEach((J)=>J(G,L))}_equal(F,G){if(Object.is(F,G))return!0;if(typeof F!==typeof G)return!1;if(typeof F!=="object"||F===null||G===null)return!1;return j(F,G)}trackDependency(){if(K)K.addDependency(this)}toJSON(){return{id:this._id,value:this._value}}dispose(){this._disposed=!0,this._subscribers.clear()}isDisposed(){return this._disposed}}class R{_value;_compute;_dependencies=new Set;_subscribers=new Set;_depUnsubs=new Map;_dirty=!0;_disposed=!1;constructor(F){this._compute=F,this._value=void 0,this._recompute()}get value(){if(this._dirty)this._recompute();return this.trackDependency(),this._value}get(){return this.value}subscribe(F){return this._subscribers.add(F),()=>this._subscribers.delete(F)}_recompute(){let F=new Set(this._dependencies);this._dependencies.clear();let G=K;K={addDependency:(J)=>{this._dependencies.add(J)}};try{this._value=this._compute(),this._dirty=!1}finally{K=G}F.forEach((J)=>{if(!this._dependencies.has(J)){let H=this._depUnsubs.get(J);if(H)H(),this._depUnsubs.delete(J)}}),this._dependencies.forEach((J)=>{if(!F.has(J)){let H=J.subscribe(()=>{this._dirty=!0,this._notifySubscribers()});this._depUnsubs.set(J,H)}})}_notifySubscribers(){if(Y>0){x.add(this);return}this.notify()}notify(){let F=this._dirty?void 0:this._value;if(this._dirty)this._recompute();let G=this._value;this._subscribers.forEach((L)=>L(G,F??G))}trackDependency(){if(K)K.addDependency(this)}dispose(){this._disposed=!0,this._depUnsubs.forEach((F)=>F()),this._depUnsubs.clear(),this._dependencies.clear(),this._subscribers.clear()}isDisposed(){return this._disposed}}class O{_fn;_cleanup;_dependencies=new Set;_depUnsubs=new Map;_id;_active=!0;_disposed=!1;constructor(F){this._fn=F,this._id=++S,this._cleanup=void 0,this._run()}_run(){if(!this._active||this._disposed)return;if(this._cleanup)this._cleanup(),this._cleanup=void 0;let F=new Set(this._dependencies);this._dependencies.clear(),M.push(this),K=this;try{this._cleanup=this._fn()}finally{M.pop(),K=M[M.length-1]||null}F.forEach((G)=>{if(!this._dependencies.has(G)){let L=this._depUnsubs.get(G);if(L)L(),this._depUnsubs.delete(G)}}),this._dependencies.forEach((G)=>{if(!F.has(G)){let L=G.subscribe(()=>this.notify());this._depUnsubs.set(G,L)}})}addDependency(F){this._dependencies.add(F)}notify(){this._run()}pause(){this._active=!1}resume(){this._active=!0,this._run()}dispose(){if(this._cleanup)this._cleanup();this._disposed=!0,this._depUnsubs.forEach((F)=>F()),this._depUnsubs.clear(),this._dependencies.clear()}isDisposed(){return this._disposed}}function p(F){return new O(F)}function v(F,G){let L=Array.isArray(F)?F:[F],J=[],H=L.map((Q)=>Q.get());return L.forEach((Q)=>{J.push(Q.subscribe(()=>{let Z=L.map(($)=>$.get()),_=H;H=[...Z],G(Array.isArray(F)?Z:Z[0],Array.isArray(F)?_:_[0])}))}),()=>J.forEach((Q)=>Q())}class T{_runes=new Map;set(F,G){let L=this._runes.get(F);if(L)return L.set(G),L;let J=new C(G);return this._runes.set(F,J),J}get(F){return this._runes.get(F)}has(F){return this._runes.has(F)}delete(F){return this._runes.delete(F)}clear(){this._runes.clear()}toJSON(){let F={};return this._runes.forEach((G,L)=>{F[L]=G.get()}),F}fromJSON(F){Object.entries(F).forEach(([G,L])=>{if(this._runes.has(G))this._runes.get(G).set(L);else this.set(G,L)})}dispose(){this._runes.forEach((F)=>{if("dispose"in F&&typeof F.dispose==="function")F.dispose()}),this._runes.clear()}isDisposed(){return this._runes.size===0}}var U=null,z=!1;function k(){if(!X()||z)return;z=!0,U=document.createElement("div"),U.id="gospa-devtools",U.innerHTML=`
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
	`,document.body.appendChild(U),U.querySelector("#gospa-devtools-close")?.addEventListener("click",()=>{U?.remove(),U=null,z=!1});let G=U.querySelectorAll("#gospa-devtools-tabs button");G.forEach((Z)=>{Z.addEventListener("click",()=>{G.forEach((W)=>W.classList.remove("active")),Z.classList.add("active");let _=Z.getAttribute("data-tab");U?.querySelectorAll(".gospa-devtools-tab-content")?.forEach((W)=>{W.style.display=W.id===`gospa-devtools-${_}`?"block":"none"})})});let L=U.querySelector("#gospa-devtools-header"),J=!1,H=0,Q=0;L?.addEventListener("mousedown",(Z)=>{let _=Z;J=!0,H=_.clientX-(U?.offsetLeft||0),Q=_.clientY-(U?.offsetTop||0)}),document.addEventListener("mousemove",(Z)=>{if(J&&U){let _=Z;U.style.left=`${_.clientX-H}px`,U.style.top=`${_.clientY-Q}px`,U.style.right="auto",U.style.bottom="auto"}}),document.addEventListener("mouseup",()=>{J=!1}),console.log("%c[GoSPA DevTools] Panel initialized","color: #e94560")}function c(){if(!U||!X())return;let F=U.querySelector("#gospa-devtools-components");if(F){let J=window.__GOSPA__?.components;if(J){let H='<div class="gospa-devtools-section">';H+='<div class="gospa-devtools-section-title">Components</div>';for(let[Q,Z]of J){let _=Z.states?Array.from(Z.states.keys()):[];H+=`<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${Q}</span>
					<span class="gospa-devtools-value">(${_.length} states)</span>
				</div>`}H+="</div>",F.innerHTML=H}}let G=U.querySelector("#gospa-devtools-state");if(G){let J=window.__GOSPA__?.globalState;if(J){let H='<div class="gospa-devtools-section">';H+='<div class="gospa-devtools-section-title">Global State</div>';let Q=J.toJSON?J.toJSON():{};for(let[_,$]of Object.entries(Q)){let W=typeof $==="object"?JSON.stringify($):String($);H+=`<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${_}:</span>
					<span class="gospa-devtools-value">${W}</span>
				</div>`}H+="</div>";let Z=window.__GOSPA_STORES__;if(Z){H+='<div class="gospa-devtools-section">',H+='<div class="gospa-devtools-section-title">Reactive Stores</div>';for(let[_,$]of Object.entries(Z)){let W=typeof $==="object"?JSON.stringify($):String($);H+=`<div class="gospa-devtools-item">
            <span class="gospa-devtools-key">${_}:</span>
            <span class="gospa-devtools-value">${W}</span>
          </div>`}H+="</div>"}G.innerHTML=H}}let L=U.querySelector("#gospa-devtools-performance");if(L){let J='<div class="gospa-devtools-section">';if(J+='<div class="gospa-devtools-section-title">Performance Metrics</div>',"memory"in performance&&performance.memory){let Q=performance.memory,Z=(Q.usedJSHeapSize/1024/1024).toFixed(2),_=(Q.totalJSHeapSize/1024/1024).toFixed(2);J+=`<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Heap Used</span>
				<span class="gospa-devtools-metric-value">${Z}MB / ${_}MB</span>
			</div>`}let H=performance.getEntriesByType("measure");if(H.length>0){let Q=H[H.length-1];J+=`<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Last Measure</span>
				<span class="gospa-devtools-metric-value">${Q.name}: ${Q.duration.toFixed(2)}ms</span>
			</div>`}J+="</div>",L.innerHTML=J}}function r(){if(!X())return;if(U)U.remove(),U=null,z=!1;else k()}function X(){return typeof window<"u"&&window.__GOSPA_DEV__!==!1}function A(...F){if(!X())return{with:()=>{}};let G=!0,L=[],J=()=>F.map((Q)=>typeof Q==="function"?Q():Q),H=(Q)=>{let Z=J();console.log(`%c[${Q}]`,"color: #888",...Z),L.forEach((_)=>_(Q,Z))};return new O(()=>{if(J(),G)G=!1,H("init");else H("update")}),{with:(Q)=>{L.push(Q)}}}A.trace=(F)=>{if(!X())return;console.log(`%c[trace]${F?` ${F}`:""}`,"color: #666; font-style: italic")};
export{k as ta,c as ua,r as va,g as wa,h as xa,m as ya,E as za,b as Aa,C as Ba,R as Ca,O as Da,p as Ea,v as Fa,T as Ga};
