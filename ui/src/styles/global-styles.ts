import { createGlobalStyle } from 'styled-components';
import './main.css';
export const GlobalStyle = createGlobalStyle`



.scroll-container {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  overflow: hidden;
  will-change: transform;
}

li::marker  {
    display: inline-block;
    vertical-align: top;
  }

.ga-pill-box__item--select {
    cursor: pointer;
    display: inline-flex;
    width: fit-content;

}
.ga-pill-box__item--select::before {
    content: attr(title);
    pointer-events: none;
}
.ͼ42 {
  background-color: transparent !important;
}
.shape-box{
    border-bottom-left-radius: 100% 250px;
    border-bottom-right-radius: 100% 250px;
    width: calc(100% + 100px) ;
    margin-left: -50px;
    overflow: hidden;
}

.ga-pill-box__item--select select {
    padding: 0;
    margin: 0;
    border: none;
    opacity: 0;
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    left: 0;
    cursor: pointer;
}


.ga-icon__add-circle {
  background-position: 8px center;
  background-image: url(data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyMCIgaGVpZ2h0PSIyMCIgZmlsbD0ibm9uZSIgdmlld0JveD0iMCAwIDIwIDIwIj48ZyBmaWxsPSIjMmYzMjM3Ij48cGF0aCBkPSJNOS41IDYuNWEuNS41IDAgMSAxIDEgMHYzaDIuOTk4YS41LjUgMCAwIDEgMCAxLjAwMUgxMC41djNhLjUuNSAwIDAgMS0xIDB2LTNoLTNhLjUuNSAwIDEgMSAwLTEuMDAxaDN6Ii8+PHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBkPSJNMTAgM2E3IDcgMCAxIDEgMCAxNCA3IDcgMCAwIDEgMC0xNHptMCAxYTYgNiAwIDEgMCAwIDEyIDYgNiAwIDAgMCAwLTEyeiIgY2xpcC1ydWxlPSJldmVub2RkIi8+PC9nPjwvc3ZnPg==);
}
/*  */
.message-container > .day-0 ~ .day-0 > .date-header {
  display: none;
}
.message-container > .day-1 ~ .day-1 > .date-header {
  display: none;
}
.message-container > .day-2 ~ .day-2 > .date-header {
  display: none;
}
.message-container > .day-3 ~ .day-3 > .date-header {
  display: none;
}
.message-container > .day-4 ~ .day-4 > .date-header {
  display: none;
}
.message-container > .day-5 ~ .day-5 > .date-header {
  display: none;
}
.message-container > .day-6 ~ .day-6 > .date-header {
  display: none;
}
.message-container > .day-7 ~ .day-7 > .date-header {
  display: none;
}
.message-container > .day-8 ~ .day-8 > .date-header {
  display: none;
}
.message-container > .day-9 ~ .day-9 > .date-header {
  display: none;
}
.message-container > .day-10 ~ .day-10 > .date-header {
  display: none;
}
.message-container > .day-11 ~ .day-11 > .date-header {
  display: none;
}
.message-container > .day-12 ~ .day-12 > .date-header {
  display: none;
}
.message-container > .day-13 ~ .day-13 > .date-header {
  display: none;
}
.message-container > .day-14 ~ .day-14 > .date-header {
  display: none;
}
.message-container > .day-15 ~ .day-15 > .date-header {
  display: none;
}
.message-container > .day-16 ~ .day-16 > .date-header {
  display: none;
}
.message-container > .day-17 ~ .day-17 > .date-header {
  display: none;
}
.message-container > .day-18 ~ .day-18 > .date-header {
  display: none;
}
.message-container > .day-19 ~ .day-19 > .date-header {
  display: none;
}
.message-container > .day-20 ~ .day-20 > .date-header {
  display: none;
}
.message-container > .day-21 ~ .day-21 > .date-header {
  display: none;
}
.message-container > .day-22 ~ .day-22 > .date-header {
  display: none;
}
.message-container > .day-23 ~ .day-23 > .date-header {
  display: none;
}
.message-container > .day-24 ~ .day-24 > .date-header {
  display: none;
}
.message-container > .day-25 ~ .day-25 > .date-header {
  display: none;
}
.message-container > .day-26 ~ .day-26 > .date-header {
  display: none;
}
.message-container > .day-27 ~ .day-27 > .date-header {
  display: none;
}
.message-container > .day-28 ~ .day-28 > .date-header {
  display: none;
}
.message-container > .day-29 ~ .day-29 > .date-header {
  display: none;
}
.message-container > .day-30 ~ .day-30 > .date-header {
  display: none;
}



body > iframe {
  display: none;
}
/* end */


@keyframes animateTop {
  25% {
    width: 100%;
    opacity: 1;
  }

  30%,
  100% {
    opacity: 0;
  }
}

/* start of react flow */
.wmde-markdown {
  font-family: "Rubik", sans-serif !important;
}

.workflow-panel-animation .react-flow__viewport {
  transition: transform 0.3s ease-in-out;
}

.workflow-node-animation .react-flow__node {
  transition: transform 0.2s ease-in-out;
}
.md-editor-toolbar-warp {
  border: none !important;
}

.react-flow__attribution {
  position: absolute;
  bottom: 0px;
  right: 0px;
}
/* end of react flow */
@keyframes fill {
    0% {
      width: 0;
    }
    100% {
      width: 100%;
    }
  }
  /* input[type="radio"]:checked + span {
  display: block; */
/* } */

.gridBody {
  fill: white
}

.monaco-editor { position: absolute !important; }

.normal-text .ͼ1 .cm-scroller {
  font-family: "Rubik", sans-serif !important;
}
.cm-content {
  white-space: pre-wrap !important;
  
}
.markdown-editor .cm-line {
  font-family: "Rubik", sans-serif;
}

.cm-line .cm-placeholder {
  opacity: 0.5;
  color: rgb(71 85 105 / var(--tw-text-opacity));
}

.monaco-editor, .monaco-editor-background {
  background: transparent !important
}

.markdown-viewer-with-uiw .wmde-markdown {
  font-family: "Rubik", sans-serif;
}

.monaco-editor .margin-view-overlays .current-line, .monaco-editor .view-overlays .current-line {
  border: none !important;
}

.cm-scroller {
  line-height: 30px !important;
}

.margin-view-overlays {
    padding-left: 10px;
}

/* hide readonly tooltip */
.monaco-editor-overlaymessage {
    display: none !important;
}

.cm-editor {
  outline: none !important;
  
}
.md-editor-content {
  border-radius: 10px;
}

.full-screen .cm-editor {
  height: 78vh !important;
}
@media (prefers-color-scheme: dark) {
  .cm-line .cm-placeholder {
    color: rgb(148 163 184 / var(--tw-text-opacity)) !important;
  }
   .cm-gutters {
    background: rgb(30 41 59) !important;
    color: rgb(100 116 139) !important;
    border: none !important;
  } 
  .cm-content {
    /* background: rgb(2 6 23) !important; */
    border: none !important;

  } 
  .monaco-editor .margin {
    background: rgb(2 6 23) !important;
  }
  .cm-content {
    caret-color: white !important
  }
  .ͼ1 .cm-cursor, .ͼ1 .cm-dropCursor {
    border-left: 1px solid white !important;
    border-right: 1px solid black !important;
  }
}
.cm-content {
    /* background: white; */
    /* padding-left: 10px !important; */
    border: none !important;
  }

  .ͼo {
    background-color: transparent !important;
  }
.md-editor-content .cm-line::selection{
  background: #c3c3c34d !important;
}

/* animate */

@keyframes rotate {
from {
  transform: rotate(0);
}

to {
  transform: rotate(360deg);
}
}

.pill-box-animator::before {
  content: "";
  display: block;
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0) 0%,
    rgba(0, 102, 204, 0.75) 50%,
    rgba(255, 255, 255, 0) 100%
);
  height: 300px;
  width: 100px;
  transform: translate(0);
  position: absolute;
  animation: rotate 1s linear forwards infinite;
  z-index: 0;
  top: 50%;
  transform-origin: top center;
}


 /* electron app bar */
 .electron .electron-titlebar { 
}





.listening {
    /* background: -moz-linear-gradient(45deg, #02e1ba 0%, #26c9f2 29%, #d911f2 66%, #ffa079 100%);
    background: -webkit-linear-gradient(45deg, #02e1ba 0%,#26c9f2 29%,#d911f2 66%,#ffa079 100%);
    background: linear-gradient(45deg, #02e1ba 0%,#26c9f2 29%,#d911f2 66%,#ffa079 100%); */
    /* background-size: 140% 140%; */
    /* -webkit-animation: Gradient 15s ease infinite; */
	/* -moz-animation: Gradient 15s ease infinite; */
	/* animation: Gradient 15s ease infinite; */
	display: flex;
	flex-direction: column;
	align-items: stretch;
	justify-content: space-evenly;
	overflow: hidden;
	position: relative; 
}

.listening::before, 
.listening::after {
	content: "";
	width: 20vmax;
	height: 20vmax;
	position: absolute;
	background: rgba(255, 255, 255, 0.1);
	/* left: 0vmin; */
	bottom: -27vmin;
    right: 0vmin;
    left: -18vmin;
	animation: morph 1s linear infinite alternate, spin 2s linear infinite;
	z-index: 1;
	will-change: border-radius, transform;
	transform-origin: 55% 55%;
	pointer-events: none; 
}
	
.listening::after {
  width: 20vmax;
	height: 20vmax;
	background: rgba(255, 255, 255, 0.2);
	/* left: 0vmin; */
	bottom: -27vmin;
  left: -10vmin;
    right: 0vmin;
	animation: morph 2s linear infinite alternate, spin 4s linear infinite;
	z-index: 1;
	will-change: border-radius, transform;
	transform-origin: 55% 55%;
	pointer-events: none;
}

@-webkit-keyframes Gradient {
	0% {
		background-position: 0 50%
	}
	50% {
		background-position: 100% 50%
	}
	100% {
		background-position: 0 50%
	}
}

@-moz-keyframes Gradient {
	0% {
		background-position: 0 50%
	}
	50% {
		background-position: 100% 50%
	}
	100% {
		background-position: 0 50%
	}
}

@keyframes Gradient {
	0% {
		background-position: 0 50%
	}
	50% {
		background-position: 100% 50%
	}
	100% {
		background-position: 0 50%
	}
}

@keyframes morph {
  0% {
    border-radius: 40% 60% 60% 40% / 70% 30% 70% 30%; }
  100% {
    border-radius: 40% 60%; } 
}

@keyframes spin {
  to {
    transform: rotate(1turn); 
  } 
}
	.st0{display:none;}
	.st1{display:inline;}
	.st2{opacity:0.29;}
	.st3{fill:#FFFFFF;}
	.st4{clip-path:url(#SVGID_2_);fill:#FFFFFF;}
	.st5{clip-path:url(#SVGID_4_);}
	.st6{clip-path:url(#SVGID_6_);}
	.st7{clip-path:url(#SVGID_8_);}
	.st8{clip-path:url(#SVGID_10_);}
	.st9{fill:none;}
	.st10{clip-path:url(#SVGID_12_);}
	.st11{opacity:0.7;}
	.st12{clip-path:url(#SVGID_14_);}
	.st13{opacity:0.2;}
	.st14{clip-path:url(#SVGID_16_);}
	.st15{opacity:0.3;fill:#FFFFFF;enable-background:new    ;}
`;
