/*
 * Copyright 2022 Stephen Guo (stephen.fire@gmail.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rtl

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestDecode(t *testing.T) {
	stream, err := hex.DecodeString("959c94d47a0e5d870b13de0d0477a96c9680aca69ff1ad10c095d29d9cf6b79486ca9dbe34255748ffe247d8e206420cb6d0a606a90877afa9e122fb844913ee76cfea884919d56d045308aad271dd71836ee071b883e1fab47fc920af8094d46189c19778df0c695405101303c2e07d6bf6ca8cc081f73697eee00615f2709ce8934c66b0b318d6daf865b0ee85f70b9419464a55d2cf28c93d342a4e3e1481d755e1a4c435b7d68094d41adc5545e33a1dee6e78ffa0c28dc005efe20c6ac0b2c7021478445fcf2f0ab8e9001997616f4d38947a6902d52833a1786d43c018c7631c02c3eb2b1f8094d4c16a9a755a633f70431ede2ece0c3fa492259163c0ee5a466c2fc48023c73dc07c59e97f0f1af32c8150223b47af49d2fbc971d6d5c649c87ebd2a108094d4f94d8683ca4e0a366288fb2db51d1d97f368b94fc0c22c60a13128276291f1fae10fe3df5d3449ae6815cf2b82a70f32bdd0cb468fd58209f6f66f6b43e3d87f3d153ee581368f8e034c728094d43f92af1e9dc5f97f50bdc09873b35a39875a8b3ac0dcda1aeb9cdbd35b4737ce2eb0f5aa409cbb0f0f44ecb5177a5a9bb17ce14b01ddfa76bc07b828840549cd830fe23cee720d6efb7aa5b877e8aa002867418094d40e8c0caf536326f17d3c0a8ef3e227903c5d4469c09837d34545a5bc542ef8942bbe26a249fd096ea2e41af66e686d45f9c8c02bb5c69bbbc60d33738094d4d94145af96cd4e130251c61a7e091b7e9ae8957cc071f4fb267d9827ee599487e468209d5264d09c7bb42c43e56fbe3044a05e857ccea3704e0732c5219253f9b5e8e1008094d472a33e79adb6f2cbd6e4269a21e5c68a9af43cdbc000a581b7a1eccfc08d850fe6c81d6cd4d99fa6deb21689a7d8c853fb32fcc9d7dabd5fe024ead8d850aa12dd15c073cab98fed3b0743d2b4d7ef178094d43b66603053dfda851aff3a34ed5361eb31699cfcc0804cbb8ef2514cd49b9668e9787fbca807b605af9565d2022963954b31b4aecac6eb46ffe720898094d4aa6c0124c3778e85f701976d1c810b0e59cc7bd5c00deccf3a5754120d479d335461ca563b24db7e055028f9f62a5320a5eee6e1f0e12195242c00513b5c019390236737301789db6c126df43a489781091fbae24ff544f68094d452bb70a8778e4815b2679c4ac0d67ab1553f7bdcc0b55446bd5895b6878bfd9c9e81ab83189741f08786b2643a47cfc075ea4f0773d5fed055eaa54db2518634c4388a60fb8b7aa1d976ad808918d43f92af1e9dc5f97f50bdc09873b35a39875a8b3aa0c028c8572bdcbee1d40e8c0caf536326f17d3c0a8ef3e227903c5d4469a0cb4f67f553531415d472a33e79adb6f2cbd6e4269a21e5c68a9af43cdba004ef5f668e1e8d4cd4aa6c0124c3778e85f701976d1c810b0e59cc7bd5a04f424bd846556e56d452bb70a8778e4815b2679c4ac0d67ab1553f7bdca07de5e25a337800dfd41adc5545e33a1dee6e78ffa0c28dc005efe20c6aa078f4fd291c4ad085d4c16a9a755a633f70431ede2ece0c3fa492259163a007b29b1088504a22d4f94d8683ca4e0a366288fb2db51d1d97f368b94fa009322278d1af23bad4d94145af96cd4e130251c61a7e091b7e9ae8957ca00c90ebce2acd1c06d43b66603053dfda851aff3a34ed5361eb31699cfca02dd856dff94b8d10d47a0e5d870b13de0d0477a96c9680aca69ff1ad10a0dede62369b774a7ed46189c19778df0c695405101303c2e07d6bf6ca8ca0a2463a4f6e5aca760c80a43f263e90")
	if err != nil {
		t.Fatal(err)
	}
	//
	// {
	// 	o := new(ForBenchmark0)
	// 	if err = Unmarshal(stream, o); err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	t.Logf("%v", o)
	// }

	{
		o := new(ForBenchmark0)
		if err = DecodeV2(bytes.NewBuffer(stream), o); err != nil {
			t.Fatal(err)
		}
		t.Logf("%v", o)
	}
}
