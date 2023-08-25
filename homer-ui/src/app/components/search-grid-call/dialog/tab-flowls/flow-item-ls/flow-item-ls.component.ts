import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core';

@Component({
    selector: 'app-flow-itemls',
    templateUrl: './flow-item-ls.component.html',
    styleUrls: ['./flow-item-ls.component.scss']
})
export class FlowItemlsComponent {
    @Input() txItem: any = {};
    @Input() isSimplify = true;
    @Input() idx = 0;
    @Output() click: EventEmitter<any> = new EventEmitter();
    @Output() clickItemShow: EventEmitter<any> = new EventEmitter();
    callData: Array<any>;

    ngOnInit() {
        this.initData();
    }

    initData () {
        this.callData = this.txItem ? this.txItem.call_data : [];
    }

    onClickItem(item, event) {
        this.click.emit({ item, event });
    }

    onClickItemShow() {
        this.clickItemShow.emit({index: this.idx});
    }
}
