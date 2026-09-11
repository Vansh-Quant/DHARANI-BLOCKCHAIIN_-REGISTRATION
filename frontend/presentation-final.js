(() => {
  window.acceptPresentationTransfer = async function(id){
    try {
      await request('/transfer/'+id+'/accept',{method:'POST'});
      const list=(await request('/transfer/incoming')).transfers||[];
      const t=list.find(x=>x.transfer_id===id);
      toast('Transfer completed');
      if(t?.property_id) await renderOwnerHistory(t.property_id); else await renderTransfer();
    } catch(e){toast(e.message)}
  };
})();
