var D=Object.create;var{getPrototypeOf:V,defineProperty:k,getOwnPropertyNames:g}=Object;var h=Object.prototype.hasOwnProperty;var l=(A,J,q)=>{q=A!=null?D(V(A)):{};let G=J||!A||!A.__esModule?k(q,"default",{value:A,enumerable:!0}):q;for(let x of g(A))if(!h.call(G,x))k(G,x,{get:()=>A[x],enumerable:!0});return G};var s=((A)=>typeof require<"u"?require:typeof Proxy<"u"?new Proxy(A,{get:(J,q)=>(typeof require<"u"?require:J)[q]}):A)(function(A){if(typeof require<"u")return require.apply(this,arguments);throw Error('Dynamic require of "'+A+'" is not supported')});var F=0,Y=[],C=new Set;function z(A){if(!C.has(A))C.add(A),Y.push(A)}function m(){for(let A=0;A<Y.length;A++)Y[A].notify();Y.length=0,C.clear()}function a(A){F++;try{A()}finally{if(F--,F===0)m()}}var y=new Set,f=null;if(typeof globalThis.FinalizationRegistry<"u")f=new globalThis.FinalizationRegistry((A)=>{});var b=!1;function AA(A=!0){b=A}function JA(){let A=0;for(let J of y)if(J.deref())A++;return A}function qA(){for(let A of y){let J=A.deref();if(J&&!J.isDisposed())J.dispose()}y.clear()}function P(A){return A}class v{_disposables=new Set;_disposed=!1;_parent=null;constructor(A=K){if(this._parent=A,A)A.add(this)}add(A){if(this._disposed){A.dispose();return}this._disposables.add(A)}remove(A){this._disposables.delete(A)}dispose(){if(this._disposed)return;this._disposed=!0;for(let A of this._disposables)A.dispose();if(this._disposables.clear(),this._parent)this._parent.remove(this),this._parent=null}isDisposed(){return this._disposed}run(A){let J=K;K=this;try{return A()}finally{K=J}}}var K=null;var i=0,W=null,_=[],N=!0;function R(){return W}function I(A){let J=W;return _.push(A),W=A,J}function B(){_.pop(),W=_[_.length-1]||null}class M{_fn;_cleanup;_dependencies=new Set;_depUnsubs=new Map;_id;_active=!0;_disposed=!1;constructor(A){if(this._fn=A,this._id=++i,this._cleanup=void 0,K)K.add(this);this._run()}_run(){if(!this._active||this._disposed)return;if(this._cleanup){try{this._cleanup()}catch(J){}this._cleanup=void 0}let A=new Set(this._dependencies);this._dependencies.clear(),I(this);try{this._cleanup=this._fn()}finally{B()}A.forEach((J)=>{if(!this._dependencies.has(J)){let q=this._depUnsubs.get(J);if(q)q(),this._depUnsubs.delete(J)}}),this._dependencies.forEach((J)=>{if(!A.has(J)){let q=J.subscribe(()=>this.notify());this._depUnsubs.set(J,q)}})}addDependency(A){if(N)this._dependencies.add(A)}notify(){this._run()}pause(){this._active=!1}resume(){this._active=!0,this._run()}dispose(){if(this._cleanup)this._cleanup();this._disposed=!0,this._depUnsubs.forEach((A)=>A()),this._depUnsubs.clear(),this._dependencies.clear()}isDisposed(){return this._disposed}}function LA(A){return new M(A)}function OA(A){let J=W;W=null,N=!1;try{return A()}finally{W=J,N=!0}}function QA(A,J){let q=Array.isArray(A)?A:[A],G=[],x=q.map((H)=>H.get());return q.forEach((H)=>{G.push(H.subscribe(()=>{let O=q.map((Z)=>Z.get()),Q=[...x];x=[...O],J(Array.isArray(A)?O:O[0],Array.isArray(A)?Q:Q[0])}))}),()=>G.forEach((H)=>H())}function X(A,J){if(A===J)return!0;if(typeof A!==typeof J)return!1;if(typeof A!=="object"||A===null||J===null)return!1;if(Array.isArray(A)&&Array.isArray(J)){if(A.length!==J.length)return!1;for(let x=0;x<A.length;x++)if(!X(A[x],J[x]))return!1;return!0}if(A instanceof Date&&J instanceof Date)return A.getTime()===J.getTime();if(A instanceof Set&&J instanceof Set){if(A.size!==J.size)return!1;for(let x of A)if(!J.has(x))return!1;return!0}if(A instanceof Map&&J instanceof Map){if(A.size!==J.size)return!1;for(let[x,H]of A)if(!J.has(x)||!X(H,J.get(x)))return!1;return!0}if(Array.isArray(A)!==Array.isArray(J))return!1;let q=Object.keys(A),G=Object.keys(J);if(q.length!==G.length)return!1;for(let x of q){if(!Object.prototype.hasOwnProperty.call(J,x))return!1;if(!X(A[x],J[x]))return!1}return!0}function T(A,J,q=!1){if(Object.is(A,J))return!0;if(!q)return!1;if(typeof A!==typeof J)return!1;if(typeof A!=="object"||A===null||J===null)return!1;return X(A,J)}var r=0;class U{_value;_id;_subscribers=[];_sv=0;_disposed=!1;_hasPendingOldValue=!1;_pendingOldValue;_deep;constructor(A,J={}){this._value=A,this._id=++r,this._deep=J.deep??!1,P(this)}get value(){return this.trackDependency(),this._value}set value(A){if(this._equal(this._value,A))return;let J=this._value;this._value=A,this._notifySubscribers(J)}get(){return this.trackDependency(),this._value}set(A){this.value=A}peek(){return this._value}update(A){this.value=A(this._value)}subscribe(A){this._subscribers.push(A);let J=this._subscribers.length-1,q=this._sv;return()=>{if(this._sv===q)this._subscribers[J]=null}}_notifySubscribers(A){if(!this._hasPendingOldValue)this._hasPendingOldValue=!0,this._pendingOldValue=A;if(F>0){z(this);return}this.notify(A)}notify(A){let J=this._value,q=this._hasPendingOldValue?this._pendingOldValue:A!==void 0?A:J;this._hasPendingOldValue=!1,this._pendingOldValue=void 0;let G=this._subscribers;for(let x=0;x<G.length;x++){let H=G[x];if(H)H(J,q)}}_equal(A,J){return T(A,J,this._deep)}trackDependency(){if(W)W.addDependency(this)}toJSON(){return{id:this._id,value:this._value}}dispose(){this._disposed=!0,this._sv++,this._subscribers.length=0}isDisposed(){return this._disposed}}function UA(A,J){return new U(A,J)}class S{_value;_compute;_dependencies=new Set;_subscribers=new Set;_depUnsubs=new Map;_dirty=!0;_disposed=!1;constructor(A){this._compute=A,this._value=void 0,this._recompute()}get value(){if(this._dirty)this._recompute();return this.trackDependency(),this._value}get(){return this.value}subscribe(A){return this._subscribers.add(A),()=>this._subscribers.delete(A)}_recompute(){let A=new Set(this._dependencies);this._dependencies.clear(),I({addDependency:(q)=>{this._dependencies.add(q)}});try{this._value=this._compute(),this._dirty=!1}finally{B()}this._dependencies.forEach((q)=>{if(!A.has(q)){let G=q.subscribe(()=>{this._dirty=!0,this._notifySubscribers()});this._depUnsubs.set(q,G)}}),A.forEach((q)=>{if(!this._dependencies.has(q)){let G=this._depUnsubs.get(q);if(G)G(),this._depUnsubs.delete(q)}})}_notifySubscribers(){if(F>0){z(this);return}this.notify()}notify(){let A=this._value;if(this._dirty)this._recompute();this._subscribers.forEach((J)=>J(this._value,A))}trackDependency(){let A=R();if(A)A.addDependency(this)}dispose(){this._disposed=!0,this._depUnsubs.forEach((A)=>A()),this._depUnsubs.clear(),this._dependencies.clear(),this._subscribers.clear()}isDisposed(){return this._disposed}}function YA(A){return new S(A)}class d{_runes=new Map;_disposed=!1;set(A,J,q){if(this._disposed)throw Error("Cannot set on a disposed StateMap");let G=this._runes.get(A);if(G)return G.set(J),G;let x=new U(J,q);return this._runes.set(A,x),x}get(A){return this._runes.get(A)}has(A){return this._runes.has(A)}delete(A){return this._runes.delete(A)}clear(){this._runes.clear()}toJSON(){let A={};return this._runes.forEach((J,q)=>{A[q]=J.peek()}),A}fromJSON(A,J){Object.entries(A).forEach(([q,G])=>{if(this._runes.has(q))this._runes.get(q).set(G);else this.set(q,G,J)})}dispose(){this._runes.forEach((A)=>{if("dispose"in A&&typeof A.dispose==="function")A.dispose()}),this._runes.clear(),this._disposed=!0}isDisposed(){return this._disposed}}var L=null,w=!1;function u(){if(!$()||w)return;w=!0,L=document.createElement("div"),L.id="gospa-devtools",L.innerHTML=`
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
	`,document.body.appendChild(L),L.querySelector("#gospa-devtools-close")?.addEventListener("click",()=>{L?.remove(),L=null,w=!1});let J=L.querySelectorAll("#gospa-devtools-tabs button");J.forEach((O)=>{O.addEventListener("click",()=>{J.forEach((j)=>j.classList.remove("active")),O.classList.add("active");let Q=O.getAttribute("data-tab");L?.querySelectorAll(".gospa-devtools-tab-content")?.forEach((j)=>{j.style.display=j.id===`gospa-devtools-${Q}`?"block":"none"})})});let q=L.querySelector("#gospa-devtools-header"),G=!1,x=0,H=0;q?.addEventListener("mousedown",(O)=>{let Q=O;G=!0,x=Q.clientX-(L?.offsetLeft||0),H=Q.clientY-(L?.offsetTop||0)}),document.addEventListener("mousemove",(O)=>{if(G&&L){let Q=O;L.style.left=`${Q.clientX-x}px`,L.style.top=`${Q.clientY-H}px`,L.style.right="auto",L.style.bottom="auto"}}),document.addEventListener("mouseup",()=>{G=!1}),console.log("%c[GoSPA DevTools] Panel initialized","color: #e94560")}function sA(){if(!L||!$())return;let A=L.querySelector("#gospa-devtools-components");if(A){let G=window.__GOSPA__?.components;if(G){let x='<div class="gospa-devtools-section">';x+='<div class="gospa-devtools-section-title">Components</div>';for(let[H,O]of G){let Q=O.states?Array.from(O.states.keys()):[];x+=`<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${H}</span>
					<span class="gospa-devtools-value">(${Q.length} states)</span>
				</div>`}x+="</div>",A.innerHTML=x}}let J=L.querySelector("#gospa-devtools-state");if(J){let G=window.__GOSPA__?.globalState;if(G){let x='<div class="gospa-devtools-section">';x+='<div class="gospa-devtools-section-title">Global State</div>';let H=G.toJSON?G.toJSON():{};for(let[Q,Z]of Object.entries(H)){let j=typeof Z==="object"?JSON.stringify(Z):String(Z);x+=`<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${Q}:</span>
					<span class="gospa-devtools-value">${j}</span>
				</div>`}x+="</div>";let O=window.__GOSPA_STORES__;if(O){x+='<div class="gospa-devtools-section">',x+='<div class="gospa-devtools-section-title">Reactive Stores</div>';for(let[Q,Z]of Object.entries(O)){let j=typeof Z==="object"?JSON.stringify(Z):String(Z);x+=`<div class="gospa-devtools-item">
            <span class="gospa-devtools-key">${Q}:</span>
            <span class="gospa-devtools-value">${j}</span>
          </div>`}x+="</div>"}J.innerHTML=x}}let q=L.querySelector("#gospa-devtools-performance");if(q){let G='<div class="gospa-devtools-section">';if(G+='<div class="gospa-devtools-section-title">Performance Metrics</div>',"memory"in performance&&performance.memory){let H=performance.memory,O=(H.usedJSHeapSize/1024/1024).toFixed(2),Q=(H.totalJSHeapSize/1024/1024).toFixed(2);G+=`<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Heap Used</span>
				<span class="gospa-devtools-metric-value">${O}MB / ${Q}MB</span>
			</div>`}let x=performance.getEntriesByType("measure");if(x.length>0){let H=x[x.length-1];G+=`<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Last Measure</span>
				<span class="gospa-devtools-metric-value">${H.name}: ${H.duration.toFixed(2)}ms</span>
			</div>`}G+="</div>",q.innerHTML=G}}function tA(){if(!$())return;if(L)L.remove(),L=null,w=!1;else u()}function $(){return typeof window<"u"&&window.__GOSPA_DEV__!==!1}function E(...A){if(!$())return{with:()=>{}};let J=!0,q=[],G=()=>A.map((H)=>typeof H==="function"?H():H),x=(H)=>{let O=G();console.log(`%c[${H}]`,"color: #888",...O),q.forEach((Q)=>Q(H,O))};return new M(()=>{if(G(),J)J=!1,x("init");else x("update")}),{with:(H)=>{q.push(H)}}}E.trace=(A)=>{if(!$())return;console.log(`%c[trace]${A?` ${A}`:""}`,"color: #666; font-style: italic")};function p(A){if(!$())return{end:()=>{}};let J=performance.now();return{end:()=>{let q=performance.now()-J;console.log(`%c[timing] ${A}: ${q.toFixed(2)}ms`,"color: #0a0")}}}function c(A){if(!$())return;if("memory"in performance&&performance.memory){let q=(performance.memory.usedJSHeapSize/1024/1024).toFixed(2);console.log(`%c[memory] ${A}: ${q}MB`,"color: #a0a")}}function o(...A){if(!$())return;console.log("%c[debug]","color: #888",...A)}function n(A,J){if(!$())return{log:()=>{},dispose:()=>{}};console.log(`%c[inspector] ${A} created`,"color: #08f");let q=J.subscribe((G)=>{console.log(`%c[${A}]`,"color: #08f",G)});return{log:()=>{console.log(`%c[${A}]`,"color: #08f",J.get())},dispose:()=>{q(),console.log(`%c[inspector] ${A} disposed`,"color: #888")}}}
export{l as Ba,s as Ca,a as Da,AA as Ea,JA as Fa,qA as Ga,v as Ha,N as Ia,R as Ja,M as Ka,LA as La,OA as Ma,QA as Na,U as Oa,UA as Pa,S as Qa,YA as Ra,d as Sa,u as Ta,sA as Ua,tA as Va,$ as Wa,E as Xa,p as Ya,c as Za,o as _a,n as $a};
