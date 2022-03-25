
* chicken / egg problem with app ID? if it's always (?) required for extrinsics then what should it be for `create_application_key`? `0` seems to work so not sure if it's ignored ...?
* Determine logging strategy / infra
* Seeing strangeness on extrinsic failure. `em.api.RPC.Chain.GetBlock` is not returning data as expected - `func (e *Extrinsic) UnmarshalJSON(bz []byte) error` seems to be expecting `string` data?
* Also seeing strangeness trying to process events in the success case? Seeing `unable to decode field 1 event #3 with EventID [0 0], field System_ExtrinsicSuccess: type *types.DispatchInfo does not support Decodeable interface and could not be decoded field by field, error: unknown DispatchClass enum: 84`
  * Best guess is that the centrifuge library doesn't handle `DispatchResultWithPostInfo` as opposed to plain `DispatchResult`.
  * Also is correct that `DispatchInfo` does not have an explicit decoder so that would be the place to start - i.e. see what's actually being returned there. But that needs to be done in the fork of the centrifuge library.