import {useRef,useState} from 'react';
import {Button,Tooltip} from '../../ui';

export function ConfigurationHelp({label,content}:{label:string;content:string}){
 const [open,setOpen]=useState(false);
 const pointerWasOpen=useRef<boolean|null>(null);
 return <Tooltip content={content} open={open} onOpenChange={setOpen}>
  <Button type="button" variant="ghost" size="sm" aria-label={`${label}说明`}
   onPointerDownCapture={()=>{pointerWasOpen.current=open}}
   onPointerCancel={()=>{pointerWasOpen.current=null}}
   onClick={e=>{e.preventDefault();setOpen(!(e.detail===0?open:pointerWasOpen.current??open));pointerWasOpen.current=null}}>?</Button>
 </Tooltip>;
}
