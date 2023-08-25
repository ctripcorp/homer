
import { ChangeDetectorRef } from '@angular/core';
import {
    Component,
    OnInit,
    ViewChild,
    Output,
    EventEmitter,
    HostListener,
    AfterViewInit,
    Input,
    OnDestroy,
    ChangeDetectionStrategy
} from '@angular/core';

@Component({
    selector: 'app-modal-resizable',
    templateUrl: './modal-resizable.component.html',
    styleUrls: ['./modal-resizable.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ModalResizableComponent implements OnInit, AfterViewInit, OnDestroy {
    static ZIndex = 12;
    _content;
    _noLayout = false;
    @ViewChild('layerZIndex', { static: false }) layerZIndex;
    @ViewChild('containerWindow', { static: false }) containerWindow;
    @ViewChild('inWindow', { static: false }) inWindow;
    @ViewChild('outWindow', { static: false }) outWindow;
    _headerColor: string;
    @Input() title: string;

    @Input()
    set headerColor(val: string) {
        this._headerColor = val;
        setTimeout(() => {
            this.cdr.detectChanges();
        }, 35);
    }
    get headerColor(): string {
        return this._headerColor;
    }
    @Input() width = 700;
    @Input() height = 600;
    @Input() minWidth = 300;
    @Input() minHeight = 300;
    @Input() mouseEventData = null;
    @Input() isBrowserWindow = false;
    @Input() startZIndex = 0;
    @Input() isFullPage = false;
    @Output() close: EventEmitter<any> = new EventEmitter();
    @Output() browserWindow: EventEmitter<any> = new EventEmitter();
    __isBrowserWindow = false;

    constructor(private cdr: ChangeDetectorRef) {
        this.cdr.detach();
    }

    ngOnInit() {
        this.cdr.detectChanges();
    }

    onFullPage() {
        this.isFullPage = !this.isFullPage;
        setTimeout(() => {
            window.dispatchEvent(new Event('resize'));
        }, 100);
        this.cdr.detectChanges();
    }

    @HostListener('document:keydown.escape', ['$event']) onKeydownHandler(event: KeyboardEvent) {
        event.preventDefault();
        event.stopPropagation();
        this.onClose();
    }
    ngAfterViewInit() {
        /* attache window to native document.body */
        document.body.appendChild(this.layerZIndex.nativeElement);

        /* clculation max windows size */
        const maxWidth = window.innerWidth * 0.95;
        const maxHeight = window.innerHeight * 0.8;
        this.minWidth = Math.min(this.minWidth, maxWidth);
        this.minHeight = Math.min(this.minWidth, maxHeight);
        (s => {
            s.minWidth = Math.min(maxWidth, Math.max(this.minWidth, this.width)) + 'px';
            s.minHeight = Math.min(maxHeight, Math.max(this.minHeight, this.height)) + 'px';
            s.width = Math.min(this.width, maxWidth) + 'px';
            // s.height = Math.min(this.height, maxHeight) + 'px';
        })(this.containerWindow.nativeElement.style);

        if (this.mouseEventData) {
            const mouseOffset = 50; // 50px inside window
            this.setPositionWindow({
                x: this.mouseEventData.clientX - mouseOffset,
                y: this.mouseEventData.clientY - mouseOffset
            });
            this.mouseEventData.focus = () => this.onFocus();
        }
        this.onFocus();
        if (this.isBrowserWindow) {
            this.newWindow();
        }
    }
    setPositionWindow({ x = 0, y = 0 }) {
        const el = this.containerWindow.nativeElement;
        const positionLocal = this.getTranslatePosition(el);
        const positionGlobal = el.getBoundingClientRect();
        x = Math.min(window.innerWidth - positionGlobal.width, x);
        y = Math.min(window.innerHeight - positionGlobal.height, y);
        positionLocal.x -= positionGlobal.left;
        positionLocal.y -= positionGlobal.top;
        positionLocal.x = Math.round(positionLocal.x + x);
        positionLocal.y = Math.round(positionLocal.y + y);

        el.style.transform = `translate3d(${positionLocal.x}px, ${positionLocal.y}px, 0px)`;
    }
    onClose() {
        this.close.emit();
        this.cdr.detectChanges();
    }

    onFocus() {
        this.layerZIndex.nativeElement.style.zIndex = '' + ((ModalResizableComponent.ZIndex += 2) + this.startZIndex);
        this.cdr.detectChanges();
    }

    onResize(event: any, controlName: string) {
        this._noLayout = false;
        this.setMouseLayer(true);
        const x0 = event.clientX;
        const y0 = event.clientY;
        const winWidth = this.containerWindow.nativeElement.offsetWidth;
        const winHeight = this.containerWindow.nativeElement.offsetHeight;
        const winPosition = this.getTranslatePosition(this.containerWindow.nativeElement);

        this.containerWindow.nativeElement.classList.add('animation-off');
        const vector = {
            left: { h: -1, v: 0 },
            right: { h: 1, v: 0 },
            bottom: { h: 0, v: 1 },
            top: { h: 0, v: -1 },
            tl: { h: -1, v: -1 },
            tr: { h: 1, v: -1 },
            bl: { h: -1, v: 1 },
            br: { h: 1, v: 1 },
        };

        const size = { w: 0, h: 0 },
            vh = vector[controlName].h,
            vv = vector[controlName].v,
            position = { x: winPosition.x, y: winPosition.y };

        // window.document.body.classList.add('no-selected-text');
        window.document.body.onmousemove = evt => {
            if (vh !== 0) {
                const cX = Math.floor(Math[(vh > 0) ? 'max' : 'min'](x0 + vh * (this.minWidth - winWidth), evt.clientX));
                size.w = winWidth + vh * (cX - x0);
                position.x = Math.floor(winPosition.x + (cX - x0) / 2);
                this.containerWindow.nativeElement.style.minWidth = size.w + 'px';
            }
            if (vv !== 0) {
                const cY = Math.floor(Math[(vv > 0) ? 'max' : 'min'](y0 + vv * (this.minHeight - winHeight), evt.clientY));
                size.h = winHeight + vv * (cY - y0);
                position.y = Math.floor(winPosition.y + (cY - y0) / 2);
                this.containerWindow.nativeElement.style.minHeight = size.h + 'px';
            }

            this.containerWindow.nativeElement.style.transform = `translate3d(${position.x}px, ${position.y}px, 0px)`;
        };

        window.document.body.onmouseleave = window.document.body.onmouseup = () => {
            this.containerWindow.nativeElement.classList.remove('animation-off');
            // window.document.body.classList.remove('no-selected-text');
            this.setMouseLayer(false);
            window.document.body.onmouseleave = null;
            window.document.body.onmousemove = null;
            window.document.body.onmouseup = null;
        };
    }
    afterResize() {
        this._noLayout = true;
        window.dispatchEvent(new Event('resize'));
        this.cdr.detectChanges();
    }

    private getTranslatePosition(el): any {
        const style = window.getComputedStyle(el);
        const matrix = new WebKitCSSMatrix(style['webkitTransform']);
        return { x: matrix.m41, y: matrix.m42 };
    }

    onStartMove(event) {

        const x0 = event.clientX;
        const y0 = event.clientY;
        const winPosition = this.getTranslatePosition(this.containerWindow.nativeElement);
        winPosition.h = this.containerWindow.nativeElement.offsetHeight;

        const documentHeigth = document.body.offsetHeight;
        const top0px = (documentHeigth - winPosition.h) / -2;

        this.containerWindow.nativeElement.classList.add('animation-off');
        // window.document.body.classList.add('no-selected-text');
        this.layerZIndex.nativeElement.classList.add('active');

        window.document.body.onmousemove = evt => {
            this.setMouseLayer(true);
            const xMove = winPosition.x + (evt.clientX - x0);
            const yMove = Math.max(top0px, winPosition.y + (evt.clientY - y0));
            this.containerWindow.nativeElement.style.transform = `translate3d(${xMove}px, ${yMove}px, 0px)`;
        };

        window.document.body.onmouseleave = window.document.body.onmouseup = () => {
            this.containerWindow.nativeElement.classList.remove('animation-off');
            // window.document.body.classList.remove('no-selected-text');
            this.layerZIndex.nativeElement.classList.remove('active');
            this.setMouseLayer(false);
            window.document.body.onmouseleave = null;
            window.document.body.onmousemove = null;
            window.document.body.onmouseup = null;
        };
    }

    onWindowClose(event: any) {
        this._content.appendChild(this.inWindow.nativeElement);
        this.browserWindow.emit(this.__isBrowserWindow);

        this.onClose();
    }
    setMouseLayer(bool: boolean = true) {
        let _layer: any = document.querySelector('.mouse-layer');
        if (!_layer) {
            _layer = document.createElement('div');
            _layer.classList.add('mouse-layer');
            document.body.appendChild(_layer);
        }
        _layer.style.display = bool && !this._noLayout ? 'block' : 'none';

    }
    newWindow() {
        this.__isBrowserWindow = true;
        this._content = this.inWindow.nativeElement.parentElement;
        this.outWindow.nativeElement.appendChild(this.inWindow.nativeElement);
        this.layerZIndex.nativeElement.style.display = 'none';
        this.browserWindow.emit(this.__isBrowserWindow);
        this.cdr.detectChanges();
    }
    ngOnDestroy() {
        document.body.removeChild(this.layerZIndex.nativeElement);
    }
}
