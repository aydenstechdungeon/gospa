var P=Object.create;var{getPrototypeOf:T,defineProperty:y,getOwnPropertyNames:E}=Object;var D=Object.prototype.hasOwnProperty;var c=(A,J,O)=>{O=A!=null?P(T(A)):{};let x=J||!A||!A.__esModule?y(O,"default",{value:A,enumerable:!0}):O;for(let q of E(A))if(!D.call(x,q))y(x,q,{get:()=>A[q],enumerable:!0});return x};var u=((A)=>typeof require<"u"?require:typeof Proxy<"u"?new Proxy(A,{get:(J,O)=>(typeof require<"u"?require:J)[O]}):A)(function(A){if(typeof require<"u")return require.apply(this,arguments);throw Error('Dynamic require of "'+A+'" is not supported')});var K=0,U=new Set;function o(A){K++;try{A()}finally{if(K--,K===0){let J=[...U];U.clear(),J.forEach((O)=>O.notify())}}}var N=new Set,g=null;if(typeof globalThis.FinalizationRegistry<"u")g=new globalThis.FinalizationRegistry((A)=>{});var V=!1;function s(A=!0){V=A}function t(){let A=0;for(let J of N)if(J.deref())A++;return A}function a(){for(let A of N){let J=A.deref();if(J&&!J.isDisposed())J.dispose()}N.clear()}function S(A){return A}class h{_disposables=new Set;_disposed=!1;_parent=null;constructor(A=W){if(this._parent=A,A)A.add(this)}add(A){if(this._disposed){A.dispose();return}this._disposables.add(A)}remove(A){this._disposables.delete(A)}dispose(){if(this._disposed)return;this._disposed=!0;for(let A of this._disposables)A.dispose();if(this._disposables.clear(),this._parent)this._parent.remove(this),this._parent=null}isDisposed(){return this._disposed}run(A){let J=W;W=this;try{return A()}finally{W=J}}}var W=null;var m=0,Z=null,Y=[];function w(){return Z}function M(A){let J=Z;return Y.push(A),Z=A,J}function I(){Y.pop(),Z=Y[Y.length-1]||null}class z{_fn;_cleanup;_dependencies=new Set;_depUnsubs=new Map;_id;_active=!0;_disposed=!1;constructor(A){if(this._fn=A,this._id=++m,this._cleanup=void 0,W)W.add(this);this._run()}_run(){if(!this._active||this._disposed)return;if(this._cleanup){try{this._cleanup()}catch(J){}this._cleanup=void 0}let A=new Set(this._dependencies);this._dependencies.clear(),M(this);try{this._cleanup=this._fn()}finally{I()}A.forEach((J)=>{if(!this._dependencies.has(J)){let O=this._depUnsubs.get(J);if(O)O(),this._depUnsubs.delete(J)}}),this._dependencies.forEach((J)=>{if(!A.has(J)){let O=J.subscribe(()=>this.notify());this._depUnsubs.set(J,O)}})}addDependency(A){this._dependencies.add(A)}notify(){this._run()}pause(){this._active=!1}resume(){this._active=!0,this._run()}dispose(){if(this._cleanup)this._cleanup();this._disposed=!0,this._depUnsubs.forEach((A)=>A()),this._depUnsubs.clear(),this._dependencies.clear()}isDisposed(){return this._disposed}}function OA(A){return new z(A)}function qA(A){let J=Z;Z=null;try{return A()}finally{Z=J}}function xA(A,J){let O=Array.isArray(A)?A:[A],x=[],q=O.map((G)=>G.get());return O.forEach((G)=>{x.push(G.subscribe(()=>{let L=O.map((_)=>_.get()),Q=q;q=[...L],J(Array.isArray(A)?L:L[0],Array.isArray(A)?Q:Q[0])}))}),()=>x.forEach((G)=>G())}function X(A,J){if(A===J)return!0;if(typeof A!==typeof J)return!1;if(typeof A!=="object"||A===null||J===null)return!1;if(Array.isArray(A)&&Array.isArray(J)){if(A.length!==J.length)return!1;for(let q=0;q<A.length;q++)if(!X(A[q],J[q]))return!1;return!0}if(A instanceof Date&&J instanceof Date)return A.getTime()===J.getTime();if(A instanceof Set&&J instanceof Set){if(A.size!==J.size)return!1;for(let q of A)if(!J.has(q))return!1;return!0}if(A instanceof Map&&J instanceof Map){if(A.size!==J.size)return!1;for(let[q,G]of A)if(!J.has(q)||!X(G,J.get(q)))return!1;return!0}if(Array.isArray(A)!==Array.isArray(J))return!1;let O=Object.keys(A),x=Object.keys(J);if(O.length!==x.length)return!1;for(let q of O){if(!Object.prototype.hasOwnProperty.call(J,q))return!1;if(!X(A[q],J[q]))return!1}return!0}function R(A,J,O=!1){if(Object.is(A,J))return!0;if(!O)return!1;if(typeof A!==typeof J)return!1;if(typeof A!=="object"||A===null||J===null)return!1;return X(A,J)}var f=0;class j{_value;_id;_subscribers=new Set;_dirty=!1;_disposed=!1;_hasPendingOldValue=!1;_pendingOldValue;_deep;constructor(A,J={}){this._value=A,this._id=++f,this._deep=J.deep??!1,S(this)}get value(){return this.trackDependency(),this._value}set value(A){if(this._equal(this._value,A))return;let J=this._value;this._value=A,this._dirty=!0,this._notifySubscribers(J)}get(){return this.trackDependency(),this._value}set(A){this.value=A}peek(){return this._value}update(A){this.value=A(this._value)}subscribe(A){return this._subscribers.add(A),()=>this._subscribers.delete(A)}_notifySubscribers(A){if(!this._hasPendingOldValue)this._hasPendingOldValue=!0,this._pendingOldValue=A;if(K>0){U.add(this);return}this.notify(A)}notify(A){let J=this._value,O=this._hasPendingOldValue?this._pendingOldValue:A!==void 0?A:J;this._hasPendingOldValue=!1,this._pendingOldValue=void 0,this._subscribers.forEach((x)=>x(J,O))}_equal(A,J){return R(A,J,this._deep)}trackDependency(){if(Z)Z.addDependency(this)}toJSON(){return{id:this._id,value:this._value}}dispose(){this._disposed=!0,this._subscribers.clear()}isDisposed(){return this._disposed}}function $A(A,J){return new j(A,J)}class C{_value;_compute;_dependencies=new Set;_subscribers=new Set;_depUnsubs=new Map;_dirty=!0;_disposed=!1;constructor(A){this._compute=A,this._value=void 0,this._recompute()}get value(){if(this._dirty)this._recompute();return this.trackDependency(),this._value}get(){return this.value}subscribe(A){return this._subscribers.add(A),()=>this._subscribers.delete(A)}_recompute(){let A=new Set(this._dependencies);this._dependencies.clear(),M({addDependency:(O)=>{this._dependencies.add(O)}});try{this._value=this._compute(),this._dirty=!1}finally{I()}A.forEach((O)=>{if(!this._dependencies.has(O)){let x=this._depUnsubs.get(O);if(x)x(),this._depUnsubs.delete(O)}}),this._dependencies.forEach((O)=>{if(!A.has(O)){let x=O.subscribe(()=>{this._dirty=!0,this._notifySubscribers()});this._depUnsubs.set(O,x)}})}_notifySubscribers(){if(K>0){U.add(this);return}this.notify()}notify(){let A=this._value;if(this._dirty)this._recompute();this._subscribers.forEach((J)=>J(this._value,A))}trackDependency(){let A=w();if(A)A.addDependency(this)}dispose(){this._disposed=!0,this._depUnsubs.forEach((A)=>A()),this._depUnsubs.clear(),this._dependencies.clear(),this._subscribers.clear()}isDisposed(){return this._disposed}}function WA(A){return new C(A)}class b{_runes=new Map;set(A,J,O){let x=this._runes.get(A);if(x)return x.set(J),x;let q=new j(J,O);return this._runes.set(A,q),q}get(A){return this._runes.get(A)}has(A){return this._runes.has(A)}delete(A){return this._runes.delete(A)}clear(){this._runes.clear()}toJSON(){let A={};return this._runes.forEach((J,O)=>{A[O]=J.peek()}),A}fromJSON(A,J){Object.entries(A).forEach(([O,x])=>{if(this._runes.has(O))this._runes.get(O).set(x);else this.set(O,x,J)})}dispose(){this._runes.forEach((A)=>{if("dispose"in A&&typeof A.dispose==="function")A.dispose()}),this._runes.clear()}isDisposed(){return this._runes.size===0}}var H=null,B=!1;function p(){if(!$()||B)return;B=!0,H=document.createElement("div"),H.id="gospa-devtools",H.innerHTML=`
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
	`,document.body.appendChild(H),H.querySelector("#gospa-devtools-close")?.addEventListener("click",()=>{H?.remove(),H=null,B=!1});let J=H.querySelectorAll("#gospa-devtools-tabs button");J.forEach((L)=>{L.addEventListener("click",()=>{J.forEach((F)=>F.classList.remove("active")),L.classList.add("active");let Q=L.getAttribute("data-tab");H?.querySelectorAll(".gospa-devtools-tab-content")?.forEach((F)=>{F.style.display=F.id===`gospa-devtools-${Q}`?"block":"none"})})});let O=H.querySelector("#gospa-devtools-header"),x=!1,q=0,G=0;O?.addEventListener("mousedown",(L)=>{let Q=L;x=!0,q=Q.clientX-(H?.offsetLeft||0),G=Q.clientY-(H?.offsetTop||0)}),document.addEventListener("mousemove",(L)=>{if(x&&H){let Q=L;H.style.left=`${Q.clientX-q}px`,H.style.top=`${Q.clientY-G}px`,H.style.right="auto",H.style.bottom="auto"}}),document.addEventListener("mouseup",()=>{x=!1}),console.log("%c[GoSPA DevTools] Panel initialized","color: #e94560")}function uA(){if(!H||!$())return;let A=H.querySelector("#gospa-devtools-components");if(A){let x=window.__GOSPA__?.components;if(x){let q='<div class="gospa-devtools-section">';q+='<div class="gospa-devtools-section-title">Components</div>';for(let[G,L]of x){let Q=L.states?Array.from(L.states.keys()):[];q+=`<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${G}</span>
					<span class="gospa-devtools-value">(${Q.length} states)</span>
				</div>`}q+="</div>",A.innerHTML=q}}let J=H.querySelector("#gospa-devtools-state");if(J){let x=window.__GOSPA__?.globalState;if(x){let q='<div class="gospa-devtools-section">';q+='<div class="gospa-devtools-section-title">Global State</div>';let G=x.toJSON?x.toJSON():{};for(let[Q,_]of Object.entries(G)){let F=typeof _==="object"?JSON.stringify(_):String(_);q+=`<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${Q}:</span>
					<span class="gospa-devtools-value">${F}</span>
				</div>`}q+="</div>";let L=window.__GOSPA_STORES__;if(L){q+='<div class="gospa-devtools-section">',q+='<div class="gospa-devtools-section-title">Reactive Stores</div>';for(let[Q,_]of Object.entries(L)){let F=typeof _==="object"?JSON.stringify(_):String(_);q+=`<div class="gospa-devtools-item">
            <span class="gospa-devtools-key">${Q}:</span>
            <span class="gospa-devtools-value">${F}</span>
          </div>`}q+="</div>"}J.innerHTML=q}}let O=H.querySelector("#gospa-devtools-performance");if(O){let x='<div class="gospa-devtools-section">';if(x+='<div class="gospa-devtools-section-title">Performance Metrics</div>',"memory"in performance&&performance.memory){let G=performance.memory,L=(G.usedJSHeapSize/1024/1024).toFixed(2),Q=(G.totalJSHeapSize/1024/1024).toFixed(2);x+=`<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Heap Used</span>
				<span class="gospa-devtools-metric-value">${L}MB / ${Q}MB</span>
			</div>`}let q=performance.getEntriesByType("measure");if(q.length>0){let G=q[q.length-1];x+=`<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Last Measure</span>
				<span class="gospa-devtools-metric-value">${G.name}: ${G.duration.toFixed(2)}ms</span>
			</div>`}x+="</div>",O.innerHTML=x}}function nA(){if(!$())return;if(H)H.remove(),H=null,B=!1;else p()}function $(){return typeof window<"u"&&window.__GOSPA_DEV__!==!1}function k(...A){if(!$())return{with:()=>{}};let J=!0,O=[],x=()=>A.map((G)=>typeof G==="function"?G():G),q=(G)=>{let L=x();console.log(`%c[${G}]`,"color: #888",...L),O.forEach((Q)=>Q(G,L))};return new z(()=>{if(x(),J)J=!1,q("init");else q("update")}),{with:(G)=>{O.push(G)}}}k.trace=(A)=>{if(!$())return;console.log(`%c[trace]${A?` ${A}`:""}`,"color: #666; font-style: italic")};function v(A){if(!$())return{end:()=>{}};let J=performance.now();return{end:()=>{let O=performance.now()-J;console.log(`%c[timing] ${A}: ${O.toFixed(2)}ms`,"color: #0a0")}}}function i(A){if(!$())return;if("memory"in performance&&performance.memory){let O=(performance.memory.usedJSHeapSize/1024/1024).toFixed(2);console.log(`%c[memory] ${A}: ${O}MB`,"color: #a0a")}}function d(...A){if(!$())return;console.log("%c[debug]","color: #888",...A)}function r(A,J){if(!$())return{log:()=>{},dispose:()=>{}};console.log(`%c[inspector] ${A} created`,"color: #08f");let O=J.subscribe((x)=>{console.log(`%c[${A}]`,"color: #08f",x)});return{log:()=>{console.log(`%c[${A}]`,"color: #08f",J.get())},dispose:()=>{O(),console.log(`%c[inspector] ${A} disposed`,"color: #888")}}}
export{c as Da,u as Ea,o as Fa,s as Ga,t as Ha,a as Ia,h as Ja,w as Ka,z as La,OA as Ma,qA as Na,xA as Oa,j as Pa,$A as Qa,C as Ra,WA as Sa,b as Ta,p as Ua,uA as Va,nA as Wa,$ as Xa,k as Ya,v as Za,i as _a,d as $a,r as ab};
