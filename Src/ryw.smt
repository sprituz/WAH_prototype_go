(declare-const offset4 String)
(declare-const offset12 String)
(assert (and (and (= "key2" offset4) (= (str.++ "key" "3") offset12)) (= offset4 offset12)))
(check-sat)
