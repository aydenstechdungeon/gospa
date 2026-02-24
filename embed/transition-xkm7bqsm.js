import"./runtime-nmjc1qqb.js";var V=(j)=>j,Q=(j)=>{let z=j-1;return z*z*z+1},Y=(j)=>{return j<0.5?4*j*j*j:0.5*Math.pow(2*j-2,3)+1},W=(j)=>{return Math.sin(-13*(j+1)*Math.PI/2)*Math.pow(2,-10*j)+1},B=(j)=>{if(j<0.36363636363636365)return 7.5625*j*j;else if(j<0.7272727272727273)return 7.5625*(j-=0.5454545454545454)*j+0.75;else if(j<0.9090909090909091)return 7.5625*(j-=0.8181818181818182)*j+0.9375;else return 7.5625*(j-=0.9545454545454546)*j+0.984375};function Z(j,{delay:z=0,duration:H=400,easing:A=V}={}){let q=+getComputedStyle(j).opacity;return{delay:z,duration:H,easing:"linear",css:(w)=>`opacity: ${w*q}`}}function _(j,{delay:z=0,duration:H=400,easing:A=Q,x:q=0,y:w=0,opacity:E=0}={}){let D=getComputedStyle(j),G=+D.opacity,J=D.transform==="none"?"":D.transform;return{delay:z,duration:H,easing:"ease-out",css:(K,N)=>`
			transform: ${J} translate(${(1-K)*q}px, ${(1-K)*w}px);
			opacity: ${G-(G-E)*N}
		`}}function $(j,{delay:z=0,duration:H=400,easing:A=Q}={}){let q=getComputedStyle(j),w=+q.opacity,E=parseFloat(q.height),D=parseFloat(q.paddingTop),G=parseFloat(q.paddingBottom),J=parseFloat(q.marginTop),K=parseFloat(q.marginBottom),N=parseFloat(q.borderTopWidth),X=parseFloat(q.borderBottomWidth);return{delay:z,duration:H,easing:"ease-out",css:(L)=>`
			overflow: hidden;
			opacity: ${Math.min(L*20,1)*w};
			height: ${L*E}px;
			padding-top: ${L*D}px;
			padding-bottom: ${L*G}px;
			margin-top: ${L*J}px;
			margin-bottom: ${L*K}px;
			border-top-width: ${L*N}px;
			border-bottom-width: ${L*X}px;
		`}}function k(j,{delay:z=0,duration:H=400,easing:A=Q,start:q=0,opacity:w=0}={}){let E=getComputedStyle(j),D=+E.opacity,G=E.transform==="none"?"":E.transform,J=1-q;return{delay:z,duration:H,easing:"ease-out",css:(K,N)=>`
            transform: ${G} scale(${1-J*N});
            opacity: ${D-(D-w)*N}
        `}}function x(j,{delay:z=0,duration:H=400,easing:A=Y,amount:q=5,opacity:w=0}={}){let D=+getComputedStyle(j).opacity;return{delay:z,duration:H,easing:"ease-in-out",css:(G,J)=>`
            opacity: ${D-(D-w)*J};
            filter: blur(${J*q}px);
        `}}function F(j,{delay:z=0,duration:H=400,easing:A=V}={}){return{delay:z,duration:H,easing:"linear",css:(q,w)=>`
            opacity: ${q};
            position: absolute;
        `}}var M=new Set;function I(j,z,H){if(M.has(j))return;M.add(j);let A=z(j,H),q=A.duration||400,w=A.delay||0,E=A.css||(()=>""),D=j.getAttribute("style")||"",G=`gospa-transition-${Math.random().toString(36).substring(2,9)}`,J=`
		@keyframes ${G} {
			0% { ${E(0,1)} }
			100% { ${E(1,0)} }
		}
	`,K=document.createElement("style");K.textContent=J,document.head.appendChild(K),j.style.animation=`${G} ${q}ms ${A.easing||"linear"} ${w}ms both`,setTimeout(()=>{j.setAttribute("style",D),j.style.animation="",K.remove(),M.delete(j)},q+w)}function P(j,z,H,A){if(M.has(j))return;M.add(j);let q=z(j,H),w=q.duration||400,E=q.delay||0,D=q.css||(()=>""),G=`gospa-transition-${Math.random().toString(36).substring(2,9)}`,J=`
		@keyframes ${G} {
			0% { ${D(1,0)} }
			100% { ${D(0,1)} }
		}
	`,K=document.createElement("style");K.textContent=J,document.head.appendChild(K),j.style.animation=`${G} ${w}ms ${q.easing||"linear"} ${E}ms both`,setTimeout(()=>{K.remove(),M.delete(j),A()},w+E)}function C(j=document.body){new MutationObserver((H)=>{H.forEach((A)=>{if(A.type==="childList")A.addedNodes.forEach((q)=>{if(q.nodeType===Node.ELEMENT_NODE){let w=q;if(w.closest("[data-gospa-static]"))return;let E=w.getAttribute("data-transition-in")||w.getAttribute("data-transition");if(E){let D=R(E);if(D)I(w,D,U(w))}}}),A.removedNodes.forEach((q)=>{if(q.nodeType===Node.ELEMENT_NODE){let w=q;if(w.closest("[data-gospa-static]"))return;let E=w.getAttribute("data-transition-out")||w.getAttribute("data-transition");if(E){let D=R(E);if(D&&!M.has(w)){let G=w.cloneNode(!0);if(G.querySelectorAll("[data-bind]").forEach((J)=>J.removeAttribute("data-bind")),G.removeAttribute("data-bind"),A.previousSibling&&A.previousSibling.parentNode)A.previousSibling.parentNode.insertBefore(G,A.previousSibling.nextSibling);else if(A.target)A.target.appendChild(G);P(G,D,U(w),()=>G.remove())}}}})})}).observe(j,{childList:!0,subtree:!0})}function R(j){if(j.startsWith("fade"))return Z;if(j.startsWith("fly"))return _;if(j.startsWith("slide"))return $;if(j.startsWith("scale"))return k;if(j.startsWith("blur"))return x;if(j.startsWith("crossfade"))return F;return null}function U(j){let z=j.getAttribute("data-transition-params");if(!z)return{};try{return JSON.parse(z)}catch(H){return console.warn("Invalid transition parameters:",z),{}}}export{P as transitionOut,I as transitionIn,$ as slide,C as setupTransitions,k as scale,V as linear,_ as fly,Z as fade,W as elasticOut,Q as cubicOut,Y as cubicInOut,F as crossfade,B as bounceOut,x as blur};
export{Z as ca,_ as da,$ as ea,k as fa,x as ga,F as ha,C as ia};
